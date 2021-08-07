package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/kelseyhightower/envconfig"
	"github.com/makramkd/taskscheduler/server"
)

type config struct {
	DatabaseDSN      string   `envconfig:"DATABASE_DSN" default:"postgres://tasksched_rw:devsecret@localhost/taskdb?sslmode=disable"`
	RedisAddress     string   `envconfig:"REDIS_ADDRESS" default:"localhost:6379"`
	AvailableServers []string `envconfig:"AVAILABLE_SERVERS" default:"localhost:8081"`
}

func main() {
	c := &config{}
	if err := envconfig.Process("", c); err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	handler := server.NewTaskHandler(
		c.AvailableServers,
	)

	r := gin.Default()
	r.POST("/api/v1/tasks/create", func(c *gin.Context) {
		handler.CreateTask(c.Writer, c.Request)
	})
	r.GET("/api/v1/tasks/:task_id/status", func(c *gin.Context) {
		handler.GetTaskStatus(c.Writer, c.Request)
	})
	r.POST("/api/v1/tasks/:task_id/status", func(c *gin.Context) {
		handler.UpdateTaskStatus(c.Writer, c.Request)
	})

	r.Run(":8080")
}
