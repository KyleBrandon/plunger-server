package leaks

import (
	"context"
	"database/sql"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/google/uuid"
)

type (
	LeakResponse struct {
		ID         uuid.UUID    `json:"id"`
		CreatedAt  time.Time    `json:"created_at"`
		UpdatedAt  time.Time    `json:"updated_at"`
		DetectedAt time.Time    `json:"detected_at"`
		ClearedAt  sql.NullTime `json:"cleared_at"`
	}

	LeakStore interface {
		GetLatestLeakDetected(ctx context.Context) (database.Leak, error)
	}

	Handler struct {
		store LeakStore
	}
)
