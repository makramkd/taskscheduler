package server

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/go-redis/redis/v8"
)

func jsonString(i interface{}) string {
	b, _ := json.Marshal(i)
	return string(b)
}

type TaskOutput struct {
	AgentID string `json:"agent_id"`
	Stdout  string `json:"stdout,omitempty"`
	Stderr  string `json:"stderr,omitempty"`
}

type TaskOutputs struct {
	Outputs []*TaskOutput
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

func setMembersToOutput(members *redis.StringSliceCmd) TaskOutputs {
	ret := TaskOutputs{
		Outputs: []*TaskOutput{},
	}
	for _, member := range members.Val() {
		taskOutput := &TaskOutput{}
		json.Unmarshal([]byte(member), taskOutput)
		ret.Outputs = append(ret.Outputs, taskOutput)
	}
	return ret
}
