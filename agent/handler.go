package agent

import (
	"encoding/json"
	"net/http"

	"github.com/makramkd/taskscheduler/api"
)

type ExecutionHandler struct {
	Scheduler *Scheduler
}

func (h *ExecutionHandler) ScheduleTask(w http.ResponseWriter, r *http.Request) {
	model := &api.ScheduleTaskRequest{}
	if err := json.NewDecoder(r.Body).Decode(model); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	h.Scheduler.ScheduleTask(
		model.TaskID,
		model.Command,
		model.Frequency,
	)
}
