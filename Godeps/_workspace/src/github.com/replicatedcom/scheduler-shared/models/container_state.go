package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/replicatedcom/scheduler-shared/eventstream"
	"github.com/replicatedcom/scheduler-shared/log"
)

type ContainerState struct {
	ID           string
	Image        string
	Labels       map[string]string
	State        State
	PortBindings []PortBinding
	Node         *Node
	Time         int64
}

type State struct {
	Running    bool
	Paused     bool
	Restarting bool
	OOMKilled  bool
	Pid        int
	ExitCode   int
	Error      string
	StartedAt  time.Time
	FinishedAt time.Time
}

func (s ContainerState) ToEventstreamEvent() (*eventstream.Event, error) {
	data, err := s.ToEventstreamEventData()
	if err != nil {
		return nil, err
	}

	eventstreamEvent := &eventstream.Event{
		Type: eventstream.EventTypeContainerState,
		ID:   s.ID,
		Data: data,
		Time: s.Time,
	}

	return eventstreamEvent, nil
}

func (s ContainerState) ToEventstreamEventData() (map[string]interface{}, error) {
	b, err := json.Marshal(s)
	if err != nil {
		log.Errorf("Failed to marshal ContainerState: %v", err)
		return nil, err
	}

	data := map[string]interface{}{
		"ContainerState": string(b),
	}

	return data, nil
}

func (s *ContainerState) FromEventstreamEventData(data map[string]interface{}) error {
	b, ok := data["ContainerState"].(string)
	if !ok {
		err := errors.New("unexpected data format")
		log.Errorf("Unexpected eventstream event data format")
		return err
	}

	if err := json.Unmarshal([]byte(b), s); err != nil {
		log.Errorf("Failed to unmarshal ContainerState: %v", err)
		return err
	}

	return nil
}
