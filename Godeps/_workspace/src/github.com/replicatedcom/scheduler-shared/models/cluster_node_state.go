package models

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/replicatedcom/scheduler-shared/eventstream"
	"github.com/replicatedcom/scheduler-shared/log"
)

type ClusterNodeState struct {
	Node          Node
	Version       string
	RemoteAddress string
	IsConnected   bool
	IsInitialized bool
}

func (s ClusterNodeState) ToEventstreamEvent() (*eventstream.Event, error) {
	data, err := s.ToEventstreamEventData()
	if err != nil {
		return nil, err
	}

	eventstreamEvent := &eventstream.Event{
		Type: eventstream.EventTypeClusterNodeState,
		ID:   s.Node.ID,
		Data: data,
		Time: time.Now().UTC().Unix(),
	}

	return eventstreamEvent, nil
}

func (s ClusterNodeState) ToEventstreamEventData() (map[string]interface{}, error) {
	b, err := json.Marshal(s)
	if err != nil {
		log.Errorf("Failed to marshal ClusterNodeState: %v", err)
		return nil, err
	}

	data := map[string]interface{}{
		"ClusterNodeState": string(b),
	}

	return data, nil
}

func (s *ClusterNodeState) FromEventstreamEventData(data map[string]interface{}) error {
	b, ok := data["ClusterNodeState"].(string)
	if !ok {
		err := errors.New("unexpected data format")
		log.Errorf("Unexpected eventstream event data format")
		return err
	}

	if err := json.Unmarshal([]byte(b), s); err != nil {
		log.Errorf("Failed to unmarshal ClusterNodeState: %v", err)
		return err
	}

	return nil
}
