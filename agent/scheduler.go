package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/makramkd/taskscheduler/api"
	"github.com/pkg/errors"
)

// Scheduler manages jobs and coordinates sending their output to the
// taskscheduler server.
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
				s.runScheduledCommand(taskID, command)
			case <-s.ctx.Done():
				return
			}
		}
	}()
}

func (s *Scheduler) runScheduledCommand(taskID, command string) {
	log.Printf("running job: %s", command)

	cmnd, stdout, stderr, err := buildCommand(command, taskID)
	if err != nil {
		log.Printf("error creating command: %v", err)
		return
	}
	defer os.Remove(stdout.Name())
	defer os.Remove(stderr.Name())

	err = cmnd.Run()
	if err != nil && !errors.Is(err, &exec.ExitError{}) {
		log.Printf("failed to run command: %v", err)
		return
	}

	log.Printf("command '%s' ran with status %v", command, err)

	// go back to the beginning of the files so that we read something.
	stdout.Seek(0, io.SeekStart)
	stderr.Seek(0, io.SeekStart)

	s.finishedJobs <- &JobStatus{
		TaskID: taskID,
		Stdout: readCommandOutput(stdout),
		Stderr: readCommandOutput(stderr),
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

	req := &api.CompleteTaskRequest{
		AgentID: s.agentID,
		Stdout:  jobStatus.Stdout,
		Stderr:  jobStatus.Stderr,
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
	Stdout string
	Stderr string
}

func buildCommand(command, taskID string) (cmnd *exec.Cmd, stdout *os.File, stderr *os.File, err error) {
	commandSplit := strings.Split(command, " ")
	cmnd = exec.Command(commandSplit[0], commandSplit[1:]...)

	// redirect stdout and stderr so that we can read them later.
	stdout, err = ioutil.TempFile("", taskID)
	if err != nil {
		log.Printf("could not create temporary file for stdout: %v", err)
		return nil, nil, nil, errors.Wrap(err, "could not create temporary file for stdout")
	}
	cmnd.Stdout = stdout
	stderr, err = ioutil.TempFile("", taskID)
	if err != nil {
		log.Printf("could not create temporary file for stderr: %v", err)
		return nil, nil, nil, errors.Wrap(err, "could not create a temporary file for stderr")
	}
	cmnd.Stderr = stderr

	return
}

func readCommandOutput(r io.Reader) (output string) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		output = fmt.Sprintf("error reading output: %v", err)
		return
	}
	return string(data)
}
