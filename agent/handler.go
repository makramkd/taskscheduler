package agent

import (
	"github.com/gin-gonic/gin"
	"github.com/makramkd/taskscheduler/api"
)

type ExecutionHandler struct {
	Scheduler *Scheduler
}

func (h *ExecutionHandler) ScheduleTask(c *gin.Context) {
	model := &api.ScheduleTaskRequest{}
	c.BindJSON(model)

	h.Scheduler.ScheduleTask(
		model.TaskID,
		model.Command,
		model.Frequency,
	)
}
