package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bsm/redislock"
	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/makramkd/taskscheduler/api"
	"github.com/pkg/errors"
)

func NewTaskHandler(
	db *sql.DB,
	redisClient *redis.Client,
	availableServers []string) *TaskHandler {
	return &TaskHandler{
		db:               db,
		availableServers: availableServers,
		redisClient:      redisClient,
	}
}

// TaskHandler handles task creation and status updating and reading.
type TaskHandler struct {
	availableServers []string
	redisClient      *redis.Client
	db               *sql.DB
}

func (h *TaskHandler) CreateTask(c *gin.Context) {
	model := &api.CreateTaskRequest{}
	c.BindJSON(model)

	taskID := uuid.New().String()

	go h.scheduleTaskOnAgents(
		taskID,
		model.Command,
		model.Frequency,
	)

	c.JSON(http.StatusCreated, &api.CreateTaskResponse{
		TaskID: taskID,
	})
}

func (h *TaskHandler) GetLatestTaskExecutionOutput(c *gin.Context) {
	taskID := c.Params.ByName("task_id")

	row := h.db.QueryRow(
		`SELECT outputs, completed_at FROM task_outputs WHERE task_id = $1 ORDER BY completed_at DESC LIMIT 1`,
		taskID,
	)
	taskOutputs := &api.TaskOutputs{}
	completedAt := time.Time{}
	if err := row.Scan(taskOutputs, &completedAt); err == sql.ErrNoRows {
		c.Writer.WriteHeader(http.StatusNotFound)
		return
	} else if err != nil {
		log.Printf("unable to scan row: %v", err)
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, &api.LatestOutputResponse{
		CompletionTime: completedAt.Format(time.RFC3339),
		Outputs:        taskOutputs.Outputs,
	})
}

func (h *TaskHandler) MarkTaskComplete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	model := &api.CompleteTaskRequest{}
	c.BindJSON(model)

	taskID := c.Params.ByName("task_id")

	// synchronize all updates to the set that stores the intermediate state
	// using a distributed lock.
	locker := redislock.New(h.redisClient)
	lock, err := h.acquireLock(ctx, locker, taskID)
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
	defer lock.Release(ctx)

	key := fmt.Sprintf("%s_done", taskID)
	saddResp := h.redisClient.SAdd(ctx, key, jsonString(model))
	if saddResp.Err() != nil {
		c.AbortWithError(http.StatusInternalServerError, saddResp.Err())
	}

	setSize := h.redisClient.SCard(ctx, key)
	if setSize.Err() != nil {
		c.AbortWithError(http.StatusInternalServerError, setSize.Err())
	}

	if setSize.Val() == int64(len(h.availableServers)) {
		members := h.redisClient.SMembers(ctx, key)
		if members.Err() != nil {
			c.AbortWithError(http.StatusInternalServerError, members.Err())
		}
		if err := h.writeToDB(ctx, taskID, members); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		// reset set to empty since we want to start from scratch with a new set
		// the next time around.
		h.redisClient.Del(ctx, key)
	}

	c.Writer.WriteHeader(http.StatusOK)
}

func (h *TaskHandler) writeToDB(ctx context.Context, taskID string, members *redis.StringSliceCmd) error {
	tx, err := h.db.BeginTx(ctx, &sql.TxOptions{
		Isolation: sql.LevelDefault,
		ReadOnly:  false,
	})
	if err != nil {
		return err
	}

	_, err = tx.Exec(
		`INSERT INTO task_outputs (task_id, outputs, completed_at) VALUES ($1, $2, $3);`,
		taskID,
		setMembersToOutput(members),
		time.Now().UTC().Format(time.RFC3339),
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (h *TaskHandler) scheduleTaskOnAgents(taskID, command, frequency string) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	scheduleRequest, err := json.Marshal(&api.ScheduleTaskRequest{
		TaskID:    taskID,
		Command:   command,
		Frequency: frequency,
	})
	if err != nil {
		log.Printf("unable to marshal json: %v", err)
		return
	}

	// schedule the task on the configured servers.
	// it's possible that we might still be executing this loop
	// and start receiving MarkTaskComplete requests.
	for _, serverURL := range h.availableServers {
		_, err := client.Post(
			fmt.Sprintf("%s/api/v1/tasks/schedule", serverURL),
			"application/json",
			bytes.NewReader(scheduleRequest),
		)
		if err != nil {
			// TODO: should we retry? what's the behavior if some fail and some succeed?
			log.Printf("failed to post schedule task request to server: %s, %v", serverURL, err)
			continue
		}
	}
}

func (h *TaskHandler) acquireLock(
	ctx context.Context,
	locker *redislock.Client,
	taskID string,
) (*redislock.Lock, error) {
	var lock *redislock.Lock
	var lockErr error
	lockKey := fmt.Sprintf("lock-%s", taskID)
	// TODO: number of retries - too small?
	for i := 0; i < 10; i++ {
		lock, lockErr = locker.Obtain(ctx, lockKey, 1*time.Second, nil)
		if lockErr == redislock.ErrNotObtained {
			// someone else might be holding the lock, wait a bit and try again
			time.Sleep(5 * time.Millisecond)
		} else if lockErr != nil {
			return nil, errors.Wrap(lockErr, "error while obtaining redis lock")
		} else if lockErr == nil {
			// successfully acquired the lock
			return lock, nil
		}
	}
	return nil, errors.New("could not grab lock after retries")
}
