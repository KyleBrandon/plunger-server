package plunges

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/utils"
)

func NewHandler(store PlungeStore, sensors sensor.Sensors) *Handler {
	h := Handler{
		store,
		sensors,
	}

	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v2/plunges/status", h.handlePlungesGet)
	mux.HandleFunc("POST /v2/plunges/start", h.handlePlungesStart)
	mux.HandleFunc("PUT /v2/plunges/stop", h.handlePlungesStop)
}

func (h *Handler) handlePlungesGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handlePlungesGet")
	defer slog.Debug("<<handlePlungesGet")

	p, err := h.store.GetLatestPlunge(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "could not find a current plunge", err)
		return
	}

	plunges := databasePlungeToPlunge(p)

	utils.RespondWithJSON(w, http.StatusOK, plunges)
}

func (h *Handler) handlePlungesStart(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handlePlungesStart")
	defer slog.Debug("<<handlePlungesStart")

	// TODO: change this to be in the body
	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = DefaultPlungeDurationSeconds
	}

	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration < 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid 'duration' parameter", err)
		return
	}

	roomTemp, waterTemp, err := h.getRecentTemperatures(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	params := database.StartPlungeParams{
		StartTime:        sql.NullTime{Valid: true, Time: time.Now().UTC()},
		StartWaterTemp:   waterTemp,
		StartRoomTemp:    roomTemp,
		ExpectedDuration: int32(duration),
	}

	// Save start to database
	plunge, err := h.store.StartPlunge(r.Context(), params)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, databasePlungeToPlunge(plunge))
}

func (h *Handler) handlePlungesStop(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handlePlungesStop")
	defer slog.Debug("<<handlePlungesStop")

	p, err := h.store.GetLatestPlunge(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "No plunge timer running", nil)
		return
	}

	roomTemp, waterTemp, err := h.getRecentTemperatures(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to stop the plunge timer", err)
		return
	}

	params := database.StopPlungeParams{
		ID:           p.ID,
		EndTime:      sql.NullTime{Valid: true, Time: time.Now().UTC()},
		EndWaterTemp: waterTemp,
		EndRoomTemp:  roomTemp,
	}

	plunge, err := h.store.StopPlunge(r.Context(), params)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to stop the plunge timer", err)
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, databasePlungeToPlunge(plunge))
}

func databasePlungeToPlunge(dbPlunge database.Plunge) PlungeResponse {
	resp := PlungeResponse{
		ID:               dbPlunge.ID,
		CreatedAt:        dbPlunge.CreatedAt,
		UpdatedAt:        dbPlunge.UpdatedAt,
		StartWaterTemp:   dbPlunge.StartWaterTemp,
		StartRoomTemp:    dbPlunge.StartRoomTemp,
		EndWaterTemp:     dbPlunge.EndWaterTemp,
		EndRoomTemp:      dbPlunge.EndRoomTemp,
		Running:          dbPlunge.Running,
		ExpectedDuration: dbPlunge.ExpectedDuration,
		AvgWaterTemp:     dbPlunge.AvgWaterTemp,
		AvgRoomTemp:      dbPlunge.AvgRoomTemp,
	}

	if dbPlunge.StartTime.Valid {
		resp.StartTime = dbPlunge.StartTime.Time
	}
	if dbPlunge.EndTime.Valid {
		resp.EndTime = dbPlunge.EndTime.Time
	}

	return resp
}

func (h *Handler) getRecentTemperatures(ctx context.Context) (string, string, error) {
	temperature, err := h.store.FindMostRecentTemperatures(ctx)
	if err != nil {
		return "", "", err
	}

	waterTemp := "0.0"
	roomTemp := "0.0"
	if temperature.RoomTemp.Valid {
		roomTemp = temperature.RoomTemp.String
	}

	if temperature.WaterTemp.Valid {
		waterTemp = temperature.WaterTemp.String
	}

	return roomTemp, waterTemp, nil
}
