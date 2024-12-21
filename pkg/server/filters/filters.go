package filters

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/pkg/utils"
)

func NewHandler(store FilterStore) *Handler {
	h := Handler{
		store,
	}

	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/filters", h.handleFilterGet)
	mux.HandleFunc("POST /v1/filters/change", h.handleFilterChange)
}

func (h *Handler) handleFilterGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handlerFilterGet")
	defer slog.Debug("<<handlerFilterGet")

	dbFilters := make([]database.Filter, 0)

	// TODO: Break this up and have a separate handler for one leak vs multiple
	filter := r.URL.Query().Get("filter")
	if filter == "current" {

		dbFilter, err := h.store.GetLatestFilterChange(r.Context())
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "could not find a current filter change entry", err)
			return
		}

		dbFilters = append(dbFilters, dbFilter)

	} else {
		// TODO: Support reading of all (paginated) filters
		utils.RespondWithError(w, http.StatusNotImplemented, "read all filter change entries not supported", errors.New("read all filter change entries not supported"))
		return
	}

	response := databaseFiltersToFilters(dbFilters)

	utils.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) handleFilterChange(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handleFilterChange")
	defer slog.Debug("<<handleFilterChange")

	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid body for filter change", err)
		return
	}

	defer r.Body.Close()

	var cr ChangeFilterRequest
	if err := json.Unmarshal(body, &cr); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid body for filter change", err)
		return
	}

	args := database.ChangeFilterParams{
		ChangedAt: cr.ChangedAt,
		RemindAt:  cr.RemindAt,
	}

	dbf, err := h.store.ChangeFilter(r.Context(), args)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "Failed to save the change filter date", err)
		return
	}

	response := ChangeFilterResponse{
		ID:        dbf.ID,
		ChangedAt: dbf.ChangedAt,
		RemindAt:  dbf.RemindAt,
	}

	utils.RespondWithJSON(w, http.StatusCreated, response)
}

func databaseFiltersToFilters(dbFilters []database.Filter) []FilterResponse {
	responses := make([]FilterResponse, 0, len(dbFilters))

	for _, db := range dbFilters {
		f := FilterResponse{
			ID:        db.ID,
			CreatedAt: db.CreatedAt,
			UpdatedAt: db.UpdatedAt,
			ChangedAt: db.ChangedAt,
			RemindAt:  db.RemindAt,
		}

		responses = append(responses, f)
	}

	return responses
}
