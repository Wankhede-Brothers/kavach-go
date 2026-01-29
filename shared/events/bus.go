// Package events provides event logging for kavach hooks.
// NOTE: Each hook invocation is a SEPARATE OS process. In-memory pub/sub
// is useless across processes. Instead, events are appended to a log file
// that can be read by session end or diagnostics.
package events

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// EventType represents types of events
type EventType string

const (
	EventMemoryWrite  EventType = "memory_write"
	EventSessionStart EventType = "session_start"
	EventAgentInvoke  EventType = "agent_invoke"
	EventSkillInvoke  EventType = "skill_invoke"
	EventError        EventType = "error"
)

// Event represents a system event
type Event struct {
	Type      EventType   `json:"type"`
	Source    string      `json:"source"`
	SessionID string      `json:"session_id,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// EventBus writes events to a shared log file (cross-process safe).
type EventBus struct {
	mu sync.Mutex
}

var (
	globalBus *EventBus
	busOnce   sync.Once
)

// GetEventBus returns singleton event bus.
func GetEventBus() *EventBus {
	busOnce.Do(func() {
		globalBus = &EventBus{}
	})
	return globalBus
}

// eventLogPath returns the path to the event log file.
func eventLogPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "shared", "shared-ai", "memory", "STM", "events.jsonl")
}

// Publish appends an event to the JSONL log file.
// Safe across concurrent hook processes (append-only + short writes).
func (bus *EventBus) Publish(eventType EventType, source string, payload interface{}) {
	event := Event{
		Type:      eventType,
		Source:    source,
		Timestamp: time.Now(),
		Payload:   payload,
	}

	data, err := json.Marshal(event)
	if err != nil {
		return
	}

	bus.mu.Lock()
	defer bus.mu.Unlock()

	logPath := eventLogPath()
	os.MkdirAll(filepath.Dir(logPath), 0755)
	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "[EVENT] log write failed: %v\n", err)
		return
	}
	defer f.Close()
	f.Write(append(data, '\n'))
}

// Subscribe is a no-op in cross-process mode.
// Events are read from the JSONL log file instead.
func (bus *EventBus) Subscribe(eventType EventType) <-chan Event {
	ch := make(chan Event, 1)
	close(ch)
	return ch
}
