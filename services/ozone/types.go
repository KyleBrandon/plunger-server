package ozone

import (
	"time"

	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/google/uuid"
)

type OzoneJob struct {
	ID              uuid.UUID `json:"id"`
	Status          string    `json:"status"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	SecondsLeft     float64   `json:"seconds_left"`
	Result          string    `json:"result"`
	CancelRequested bool      `json:"cancel_requested"`
}

type Handler struct {
	manager *jobs.JobConfig
	store   jobs.JobStore
}
