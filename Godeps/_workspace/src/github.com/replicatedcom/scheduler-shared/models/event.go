package models

import (
	"time"

	"github.com/replicatedcom/scheduler-shared/eventstream"
)

const (
	EventTypeContainerStart EventType = iota
	EventTypeContainerStartSynthetic
	EventTypeContainerStop
	EventTypeContainerStopSynthetic
	EventTypeContainerRemove
	EventTypeContainerRemoveSynthetic
	EventTypeContainerPause
	EventTypeContainerPauseSynthetic
	EventTypeContainerUnpause
	EventTypeContainerUnpauseSynthetic
	EventTypePortListen
	EventTypeExec
	EventTypeCommit
)

type Event struct {
	Type  EventType
	Name  string
	Image string
	Data  map[string]interface{}
}

type EventType int

func (t EventType) String() string {
	switch t {
	case EventTypeContainerStart:
		return "container-start"
	case EventTypeContainerStartSynthetic:
		return "container-start-synthetic"
	case EventTypeContainerStop:
		return "container-stop"
	case EventTypeContainerStopSynthetic:
		return "container-stop-synthetic"
	case EventTypeContainerRemove:
		return "container-remove"
	case EventTypeContainerRemoveSynthetic:
		return "container-remove-synthetic"
	case EventTypeContainerPause:
		return "container-pause"
	case EventTypeContainerPauseSynthetic:
		return "container-pause-synthetic"
	case EventTypeContainerUnpause:
		return "container-unpause"
	case EventTypeContainerUnpauseSynthetic:
		return "container-unpause-synthetic"
	case EventTypePortListen:
		return "port-listen"
	case EventTypeExec:
		return "exec"
	case EventTypeCommit:
		return "commit"
	default:
		return "unknown"
	}
}

func (e Event) ToEventstreamEvent(id string, err error) *eventstream.Event {
	return &eventstream.Event{
		Type: eventstream.EventTypeDepends,
		ID:   id,
		Data: e.toData(),
		Err:  err,
		Time: time.Now().UTC().Unix(),
	}
}

func (e Event) toData() map[string]interface{} {
	return map[string]interface{}{
		"Type":  e.Type,
		"Name":  e.Name,
		"Image": e.Image,
		"Data":  e.Data,
	}
}
