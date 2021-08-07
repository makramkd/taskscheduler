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
	if err := json.NewDecoder(c.Request.Body).Decode(model); err != nil {
		c.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	taskID := uuid.New().String()

	go h.scheduleTaskOnAgents(
		taskID,
		model.Command,
		model.Frequency,
	)

	response := &api.CreateTaskResponse{
		TaskID: taskID,
	}
	if err := json.NewEncoder(c.Writer).Encode(response); err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}
	c.Writer.WriteHeader(http.StatusCreated)
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

	response := &api.LatestOutputResponse{
		CompletionTime: completedAt.Format(time.RFC3339),
		Outputs:        taskOutputs.Outputs,
	}
	if err := json.NewEncoder(c.Writer).Encode(response); err != nil {
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	c.Writer.WriteHeader(http.StatusOK)
}

func (h *TaskHandler) MarkTaskComplete(c *gin.Context) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	model := &api.CompleteTaskRequest{}
	if err := json.NewDecoder(c.Request.Body).Decode(model); err != nil {
		c.Writer.WriteHeader(http.StatusBadRequest)
		return
	}

	taskID := c.Params.ByName("task_id")

	// synchronize all updates to the set that stores the intermediate state
	// using a distributed lock.
	locker := redislock.New(h.redisClient)
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
			log.Printf("other error while obtaining redis lock: %v", lockErr)
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		} else if lockErr == nil {
			// successfully acquired the lock
			break
		}
	}
	defer lock.Release(ctx)

	key := fmt.Sprintf("%s_done", taskID)
	saddResp := h.redisClient.SAdd(ctx, key, jsonString(model))
	if saddResp.Err() != nil {
		log.Printf("could not add to set: %v", saddResp.Err())
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	setSize := h.redisClient.SCard(ctx, key)
	if setSize.Err() != nil {
		log.Printf("could not get set size: %v", setSize.Err())
		c.Writer.WriteHeader(http.StatusInternalServerError)
		return
	}

	if setSize.Val() == int64(len(h.availableServers)) {
		members := h.redisClient.SMembers(ctx, key)
		if members.Err() != nil {
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
		}
		if err := h.writeToDB(ctx, taskID, members); err != nil {
			log.Printf("error writing to DB: %v", err)
			c.Writer.WriteHeader(http.StatusInternalServerError)
			return
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
		`
INSERT INTO task_outputs (task_id, outputs, completed_at)
VALUES ($1, $2, $3);
`,
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
