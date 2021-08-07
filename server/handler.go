package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/makramkd/taskscheduler/api"
)

func NewTaskHandler(availableServers []string) *TaskHandler {
	return &TaskHandler{
		availableServers: availableServers,
	}
}

type TaskOutput string {

}

// TaskHandler handles task creation and status updating and reading.
type TaskHandler struct {
	availableServers []string
	taskMap          sync.Map
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	model := &api.CreateTaskRequest{}
	if err := json.NewDecoder(r.Body).Decode(model); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
}

func (h *TaskHandler) GetTaskStatus(w http.ResponseWriter, r *http.Request) {

}

func (h *TaskHandler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	model := &api.UpdateTaskRequest{}
	if err := json.NewDecoder(r.Body).Decode(model); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// TODO: improve. maybe use a library instead.
	split := strings.Split(r.URL.Path, "/")
	taskID := split[len(split)-2]

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

	// schedule the task on the configured servers
	// it's possible that we might still be executing this loop
	// and start receiving UpdateTaskStatus requests.
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
