package leaks

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/utils"
)

func NewHandler(store LeakStore) *Handler {
	h := Handler{
		store,
	}

	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/leaks", h.handlerLeakGet)
}

func (h *Handler) handlerLeakGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerGetLeak")

	dbLeaks := make([]database.Leak, 0)

	// TODO: Break this up and have a separate handler for one leak vs multiple
	filter := r.URL.Query().Get("filter")
	if filter == "current" {

		dbLeak, err := h.store.GetLatestLeakDetected(r.Context())
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "could not find the current leak event", err)
			return
		}

		dbLeaks = append(dbLeaks, dbLeak)

	} else {
		// TODO: Support reading of all (paginated) leaks
		utils.RespondWithError(w, http.StatusNotImplemented, "read all leaks not supported", errors.New("read all leaks not supported"))
		return
	}

	response := databaseLeaksToLeaks(dbLeaks)

	utils.RespondWithJSON(w, http.StatusOK, response)
}

func databaseLeaksToLeaks(dbLeaks []database.Leak) []LeakResponse {
	leaks := make([]LeakResponse, 0, len(dbLeaks))

	for _, dbLeak := range dbLeaks {

		leak := LeakResponse{
			ID:         dbLeak.ID,
			CreatedAt:  dbLeak.CreatedAt,
			UpdatedAt:  dbLeak.UpdatedAt,
			DetectedAt: dbLeak.DetectedAt,
		}

		if dbLeak.ClearedAt.Valid {
			leak.ClearedAt = dbLeak.ClearedAt
		}

		leaks = append(leaks, leak)
	}

	return leaks
}
