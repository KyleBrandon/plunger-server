package filters

import (
	"context"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/google/uuid"
)

type (
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
	}

	Handler struct {
		store FilterStore
	}
)
