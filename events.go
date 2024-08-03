package main

import (
	"encoding/json"
	"log"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
)

const (
	EVENTTYPE_LEAK = 1
)

type DbLeakEvent struct {
	EventTime     time.Time `json:"event_time"`
	PreviousState bool      `json:"previous_state"`
	CurrentState  bool      `json:"current_state"`
}

type LeakEvent struct {
	UpdatedAt    time.Time `json:"updated_at"`
	LeakDetected bool      `json:"leak_detected"`
}

func BuildLeakEventsFromEvents(events []database.Event) ([]LeakEvent, error) {
	leakEvents := make([]LeakEvent, 0, len(events))

	for _, event := range events {
		var dbLeakEvent DbLeakEvent
		err := json.Unmarshal(event.EventData, &dbLeakEvent)
		if err != nil {
			log.Printf("failed to deserialize the leak event: %v\n", err)
			return nil, err
		}

		leakEvent := LeakEvent{
			UpdatedAt:    dbLeakEvent.EventTime,
			LeakDetected: dbLeakEvent.CurrentState,
		}

		leakEvents = append(leakEvents, leakEvent)
	}

	return leakEvents, nil
}
