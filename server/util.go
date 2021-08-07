package server

import (
	"encoding/json"

	"github.com/go-redis/redis/v8"
	"github.com/makramkd/taskscheduler/api"
)

func jsonString(i interface{}) string {
	b, _ := json.Marshal(i)
	return string(b)
}

func setMembersToOutput(members *redis.StringSliceCmd) api.TaskOutputs {
	ret := api.TaskOutputs{
		Outputs: []*api.TaskOutput{},
	}
	for _, member := range members.Val() {
		taskOutput := &api.TaskOutput{}
		json.Unmarshal([]byte(member), taskOutput)
		ret.Outputs = append(ret.Outputs, taskOutput)
	}
	return ret
}
