package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/lib/pq"
	"github.com/makramkd/taskscheduler/server"
)

type config struct {
	DatabaseDSN      string   `envconfig:"DATABASE_DSN" default:"postgres://tasksched_rw:devsecret@localhost/taskdb?sslmode=disable"`
	RedisAddress     string   `envconfig:"REDIS_ADDRESS" default:"127.0.0.1:6379"`
	AvailableServers []string `envconfig:"AVAILABLE_SERVERS" default:"http://localhost:8081"`
	Port             int      `envconfig:"PORT" default:"8080"`
}

func main() {
	c := &config{}
	if err := envconfig.Process("", c); err != nil {
		log.Fatalf("failed to parse config: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Network: "tcp",
		Addr:    c.RedisAddress,
	})
	defer redisClient.Close()

	db, err := sql.Open("postgres", c.DatabaseDSN)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	handler := &server.TaskHandler{
		AvailableServers: c.AvailableServers,
		RedisClient:      redisClient,
		DB:               db,
	}

	r := gin.Default()
	r.POST("/api/v1/tasks/create", handler.CreateTask)
	r.GET("/api/v1/tasks/:task_id/latest_output", handler.GetLatestTaskExecutionOutput)
	r.POST("/api/v1/tasks/:task_id/complete", handler.MarkTaskComplete)

	r.Run(fmt.Sprintf(":%d", c.Port))
}
