package main

import (
	"context"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/kelseyhightower/envconfig"
	"github.com/makramkd/taskscheduler/agent"
)

type config struct {
	Port          int    `envconfig:"PORT" default:"8081"`
	ServerAddress string `envconfig:"TASK_SERVER_ADDRESS" default:"http://localhost:8080"`
}

func main() {
	c := &config{}
	if err := envconfig.Process("", c); err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	agentID := uuid.New().String()

	scheduler := agent.NewScheduler(c.ServerAddress, agentID, ctx)
	go scheduler.Run(ctx)

	handler := &agent.ExecutionHandler{
		Scheduler: scheduler,
	}
	r := gin.Default()
	r.POST("/api/v1/tasks/schedule", func(c *gin.Context) {
		handler.ScheduleTask(c.Writer, c.Request)
	})

	r.Run(fmt.Sprintf(":%d", c.Port))
}
