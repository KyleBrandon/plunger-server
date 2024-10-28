package filters

import (
	"context"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/google/uuid"
)

type (
	FilterResponse struct {
		ID        uuid.UUID `json:"id"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		ChangedAt time.Time `json:"changed_at"`
		RemindAt  time.Time `json:"remind_at"`
	}

	ChangeFilterRequest struct {
		ChangedAt time.Time `json:"changed_at"`
		RemindAt  time.Time `json:"remind_at"`
	}

	ChangeFilterResponse struct {
		ID        uuid.UUID `json:"id"`
		ChangedAt time.Time `json:"changed_at"`
		RemindAt  time.Time `json:"remind_at"`
	}

	FilterStore interface {
		GetFilters(ctx context.Context) ([]database.Filter, error)
		ChangeFilter(ctx context.Context, arg database.ChangeFilterParams) (database.Filter, error)
		GetLatestFilterChange(ctx context.Context) (database.Filter, error)
	}

	Handler struct {
		store FilterStore
	}
)
