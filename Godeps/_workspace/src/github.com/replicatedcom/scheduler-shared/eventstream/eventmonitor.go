package eventstream

import (
	"sync"

	uuid "github.com/nu7hatch/gouuid"
)

type EventMonitor struct {
	sync.RWMutex
	Listeners map[string]ListenerFn
}

type ListenerFn func(event *Event, removeFn func())

func NewEventMonitor() *EventMonitor {
	return &EventMonitor{
		Listeners: make(map[string]ListenerFn),
	}
}

func (m *EventMonitor) AddListener(fn ListenerFn) func() {
	m.Lock()
	defer m.Unlock()

	newID, err := uuid.NewV4()
	if err != nil {
		// this should never happen
		panic(err)
	}
	id := newID.String()

	m.Listeners[id] = fn

	return m.removeFn(id, true)
}

func (m *EventMonitor) SendEvent(event *Event) {
	m.RLock()
	defer m.RUnlock()

	for id, listener := range m.Listeners {
		listener(event, m.removeFn(id, false))
	}
}

func (m *EventMonitor) removeFn(id string, lock bool) func() {
	return func() {
		if lock {
			m.Lock()
			defer m.Unlock()
		}

		newListeners := make(map[string]ListenerFn)

		for i, listener := range m.Listeners {
			if i != id {
				newListeners[i] = listener
			}
		}

		m.Listeners = newListeners
	}
}
