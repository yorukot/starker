package tasks

import (
	"encoding/json"

	"github.com/hibiken/asynq"
)

type ServicePayload struct {
	serviceID  string
}

func NewStartServiceTask(serviceID, tmplID string) (*asynq.Task, error) {
	payload, err := json.Marshal(ServicePayload{serviceID: serviceID})
	if err != nil {
		return nil, err
	}

	return asynq.NewTask(TypeStartService, payload), nil
}
