package api

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
	Stdout  string `json:"stdout"`
	Stderr  string `json:"stderr"`
}

// ---- Agent API ----

// ScheduleTaskRequest is used to schedule a task from the agent API.
type ScheduleTaskRequest struct {
	TaskID    string `json:"task_id"`
	Command   string `json:"command"`
	Frequency string `json:"frequency"`
}
