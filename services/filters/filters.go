package filters

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/utils"
)

func NewHandler(store FilterStore) *Handler {
	h := Handler{
		store,
	}

	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v2/filter/change", h.handleFilterChange)
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
