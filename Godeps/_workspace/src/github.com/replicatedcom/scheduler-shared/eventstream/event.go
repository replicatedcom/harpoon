package eventstream

const (
	EventTypeDepends EventType = iota
	EventTypeContainerStatusChange
	EventTypeContainerState
	EventTypeSynthetic
	EventTypeClusterNodeStatusChange
	EventTypeClusterNodeState
)

type Event struct {
	Type EventType
	ID   string
	Data map[string]interface{}
	Err  error
	Time int64
}

type EventType int

func (t EventType) String() string {
	switch t {
	case EventTypeDepends:
		return "depends"
	case EventTypeContainerStatusChange:
		return "container-status-change"
	case EventTypeContainerState:
		return "container-state"
	case EventTypeSynthetic:
		return "syncthetic"
	default:
		return "unknown"
	}
}
