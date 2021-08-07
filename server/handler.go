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

// TaskHandler provides a CRUD API for managing tasks on remote machines.
type TaskHandler struct {
	AvailableServers []string
	RedisClient      *redis.Client
	DB               *sql.DB
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

	row := h.DB.QueryRow(
		`SELECT outputs, completed_at FROM task_outputs WHERE task_id = $1 ORDER BY completed_at DESC LIMIT 1`,
		taskID,
	)
	taskOutputs := &api.TaskOutputs{}
	completedAt := time.Time{}
	if err := row.Scan(taskOutputs, &completedAt); err == sql.ErrNoRows {
		c.AbortWithError(http.StatusNotFound, err)
	} else if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
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
	locker := redislock.New(h.RedisClient)
	lock, err := locker.Obtain(ctx, fmt.Sprintf("lock-%s", taskID), time.Second, &redislock.Options{
		RetryStrategy: redislock.LinearBackoff(5 * time.Millisecond),
	})
	if err != nil {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
	defer lock.Release(ctx)

	key := fmt.Sprintf("%s_done", taskID)
	saddResp := h.RedisClient.SAdd(ctx, key, jsonString(model))
	if saddResp.Err() != nil {
		c.AbortWithError(http.StatusInternalServerError, saddResp.Err())
	}

	setSize := h.RedisClient.SCard(ctx, key)
	if setSize.Err() != nil {
		c.AbortWithError(http.StatusInternalServerError, setSize.Err())
	}

	if setSize.Val() == int64(len(h.AvailableServers)) {
		members := h.RedisClient.SMembers(ctx, key)
		if members.Err() != nil {
			c.AbortWithError(http.StatusInternalServerError, members.Err())
		}
		if err := h.writeToDB(ctx, taskID, members); err != nil {
			c.AbortWithError(http.StatusInternalServerError, err)
		}

		// reset set to empty since we want to start from scratch with a new set
		// the next time around.
		h.RedisClient.Del(ctx, key)
	}
}

func (h *TaskHandler) writeToDB(ctx context.Context, taskID string, members *redis.StringSliceCmd) error {
	tx, err := h.DB.BeginTx(ctx, &sql.TxOptions{
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
	for _, serverURL := range h.AvailableServers {
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
