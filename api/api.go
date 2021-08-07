package api

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// ---- Server API ---

// CreateTaskRequest is used to create a task from the server API.
type CreateTaskRequest struct {
	Command   string `json:"command"`
	Frequency string `json:"frequency"`
}

type CreateTaskResponse struct {
	TaskID string `json:"task_id"`
}

type CompleteTaskRequest struct {
	AgentID string `json:"agent_id"`
	Stdout  string `json:"stdout,omitempty"`
	Stderr  string `json:"stderr,omitempty"`
}

type TaskOutput struct {
	AgentID string `json:"agent_id"`
	Stdout  string `json:"stdout,omitempty"`
	Stderr  string `json:"stderr,omitempty"`
}

type TaskOutputs struct {
	Outputs []*TaskOutput `json:"outputs"`
}

func (t TaskOutputs) Value() (driver.Value, error) {
	return json.Marshal(t)
}

func (t *TaskOutputs) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(b, t)
}

type LatestOutputResponse struct {
	CompletionTime string        `json:"completion_time"`
	Outputs        []*TaskOutput `json:"outputs"`
}

// ---- Agent API ----

// ScheduleTaskRequest is used to schedule a task from the agent API.
type ScheduleTaskRequest struct {
	TaskID    string `json:"task_id"`
	Command   string `json:"command"`
	Frequency string `json:"frequency"`
}
