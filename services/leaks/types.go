package leaks

import (
	"context"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/google/uuid"
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
type LeakStore interface {
	GetLatestEventByType(ctx context.Context, eventType int32) (database.Event, error)
	GetEventsByType(ctx context.Context, arg database.GetEventsByTypeParams) ([]database.Event, error)
}

type Handler struct {
	store            LeakStore
	manager          jobs.JobManager
	leakMonitorJobId uuid.UUID
}
