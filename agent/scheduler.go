package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
	"time"

	"github.com/makramkd/taskscheduler/api"
)

type Scheduler struct {
	ctx           context.Context
	finishedJobs  chan *JobStatus
	serverAddress string
	agentID       string
}

func NewScheduler(serverAddress, agentID string, ctx context.Context) *Scheduler {
	return &Scheduler{
		finishedJobs:  make(chan *JobStatus, 10000),
		serverAddress: serverAddress,
		ctx:           ctx,
		agentID:       agentID,
	}
}

func (s *Scheduler) ScheduleTask(taskID, command, frequency string) {
	go func() {
		for {
			duration, err := parseFrequency(frequency)
			if err != nil {
				log.Printf("bad frequency: %v", err)
				return
			}

			ticker := time.NewTicker(duration)
			select {
			case <-ticker.C:
				log.Printf("running command: %s", command)
				s.runScheduledCommand(taskID, command)
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

func (s *Scheduler) runScheduledCommand(taskID, command string) {
	log.Printf("running job: %s", command)

	commandSplit := strings.Split(command, " ")
	log.Printf("command split: %v", commandSplit)
	cmnd := exec.Command(commandSplit[0], commandSplit[1:]...)
	stdoutFile, err := ioutil.TempFile("", taskID)
	if err != nil {
		log.Printf("could not create temporary file for stdout: %v", err)
		return
	}
	cmnd.Stdout = stdoutFile
	stderrFile, err := ioutil.TempFile("", taskID)
	if err != nil {
		log.Printf("could not create temporary file for stderr: %v", err)
		return
	}
	cmnd.Stderr = stderrFile

	err = cmnd.Run()
	if err != nil && !errors.Is(err, &exec.ExitError{}) {
		log.Printf("failed to run command: %v", err)
		return
	}

	log.Printf("command '%s' ran with status %v", command, err)

	// go back to the beginning of the files so that we read something
	// TODO: should we just read to a string here?
	stdoutFile.Seek(0, io.SeekStart)
	stderrFile.Seek(0, io.SeekStart)

	s.finishedJobs <- &JobStatus{
		TaskID: taskID,
		Stdout: stdoutFile,
		Stderr: stderrFile,
	}
}

func (s *Scheduler) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case jobStatus := <-s.finishedJobs:
			s.updateTaskState(jobStatus)
		}
	}
}

func (s *Scheduler) updateTaskState(jobStatus *JobStatus) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	stdout, err := ioutil.ReadAll(jobStatus.Stdout)
	if err != nil {
		log.Printf("error reading stdout: %v", err)
		return // todo: should return or continue?
	}

	stderr, err := ioutil.ReadAll(jobStatus.Stderr)
	if err != nil {
		log.Printf("error reading stderr: %v", err)
		return // todo: should return or continue?
	}

	req := &api.CompleteTaskRequest{
		AgentID: s.agentID,
		Stdout:  string(stdout),
		Stderr:  string(stderr),
	}
	encoded, err := json.Marshal(req)
	if err != nil {
		log.Printf("unable to marshal json: %v", err)
		return
	}

	_, err = client.Post(
		fmt.Sprintf("%s/api/v1/tasks/%s/complete", s.serverAddress, jobStatus.TaskID),
		"application/json",
		bytes.NewReader(encoded),
	)
	if err != nil {
		log.Printf("error updating task status: %v", err)
	}
}

type JobStatus struct {
	TaskID string
	Stdout io.Reader
	Stderr io.Reader
}
