package server

import "net/http"

// TaskStatus represents the status of the task to the users of our API.
type TaskStatus string

const (
	TaskStatusInvalid          TaskStatus = "TASK_INVALID"
	TaskStatusReceived         TaskStatus = "TASK_RECEIVED"
	TaskStatusScheduled        TaskStatus = "TASK_SCHEDULED"
	TaskStatusExecuting        TaskStatus = "TASK_EXECUTING"
	TaskStatusCompletedSuccess TaskStatus = "TASK_COMPLETED_SUCCESS"
	TaskStatusCompletedError   TaskStatus = "TASK_COMPLETED_ERROR"
	TaskStatusDeleted          TaskStatus = "TASK_DELETED"
)

func NewTaskHandler(client *http.Client) *TaskHandler {
	return &TaskHandler{
		client: client,
	}
}

// TaskHandler handles task creation and status updating and reading.
// TODO inject these dependencies:
// * connection to the DB
// * client API for the agent
type TaskHandler struct {
	client *http.Client
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {

}

func (h *TaskHandler) GetTaskStatus(w http.ResponseWriter, r *http.Request) {

}

func (h *TaskHandler) UpdateTaskStatus(w http.ResponseWriter, r *http.Request) {

}
