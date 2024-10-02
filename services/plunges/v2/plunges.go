package plunges

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/utils"
	"github.com/google/uuid"
)

const DefaultPlungeDurationSeconds = "180"

// Clear the current plunge state
func (s *PlungeState) Clear() {
	s.ID = uuid.Nil
	s.Running = false
	s.WaterTempTotal = 0
	s.RoomTempTotal = 0
	s.TempReadCount = 0
}

// Start a plunger state
func (s *PlungeState) Start(id uuid.UUID) {
	s.ID = id
	s.Running = true
}

// Update the temperatures
func (s *PlungeState) UpdateAverageTemperatures(roomTemperature float64, waterTemperature float64) (float64, float64) {
	s.WaterTempTotal += waterTemperature
	s.RoomTempTotal += roomTemperature
	s.TempReadCount++
	avgWaterTemp := s.WaterTempTotal / float64(s.TempReadCount)
	avgRoomTemp := s.RoomTempTotal / float64(s.TempReadCount)

	return avgRoomTemp, avgWaterTemp
}

func NewHandler(store PlungeStore, sensors sensor.Sensors, state *PlungeState) *Handler {
	h := Handler{}

	h.store = store
	h.sensors = sensors
	h.state = state

	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v2/plunges/status", h.handlePlungesGet)
	mux.HandleFunc("POST /v2/plunges/start", h.handlePlungesStart)
	mux.HandleFunc("PUT /v2/plunges/stop", h.handlePlungesStop)
}

func (h *Handler) handlePlungesGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesGet")
	h.state.MU.Lock()
	defer h.state.MU.Unlock()

	p, err := h.store.GetLatestPlunge(r.Context())
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "could not find a current plunge", err)
		return
	}

	plunges := databasePlungeToPlunge(p)

	utils.RespondWithJSON(w, http.StatusOK, plunges)
}

func (h *Handler) handlePlungesStart(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesStart")

	durationStr := r.URL.Query().Get("duration")
	if durationStr == "" {
		durationStr = DefaultPlungeDurationSeconds
	}

	duration, err := strconv.Atoi(durationStr)
	if err != nil || duration < 0 {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid 'duration' parameter", err)
		return
	}

	roomTemp, waterTemp := h.sensors.ReadRoomAndWaterTemperature()
	if roomTemp.Err != nil || waterTemp.Err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", roomTemp.Err)
		return
	}

	params := database.StartPlungeParams{
		StartTime:        sql.NullTime{Valid: true, Time: time.Now().UTC()},
		StartWaterTemp:   fmt.Sprintf("%f", waterTemp.TemperatureF),
		StartRoomTemp:    fmt.Sprintf("%f", roomTemp.TemperatureF),
		ExpectedDuration: int32(duration),
	}

	// Save start to database
	plunge, err := h.store.StartPlunge(r.Context(), params)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	// initalize the plunge context info
	h.state.MU.Lock()
	h.state.Start(plunge.ID)
	h.state.MU.Unlock()

	utils.RespondWithJSON(w, http.StatusCreated, databasePlungeToPlunge(plunge))
}

func (h *Handler) handlePlungesStop(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesStop")

	h.state.MU.Lock()

	if !h.state.Running {
		utils.RespondWithError(w, http.StatusConflict, "No plunge timer running", nil)
		h.state.MU.Unlock()
		return
	}

	id := h.state.ID
	h.state.Clear()

	h.state.MU.Unlock()

	roomTemp, waterTemp := h.sensors.ReadRoomAndWaterTemperature()
	if roomTemp.Err != nil || waterTemp.Err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to stop the plunge timer", roomTemp.Err)
		return
	}

	params := database.StopPlungeParams{
		ID:           id,
		EndTime:      sql.NullTime{Valid: true, Time: time.Now().UTC()},
		EndWaterTemp: fmt.Sprintf("%f", waterTemp.TemperatureF),
		EndRoomTemp:  fmt.Sprintf("%f", roomTemp.TemperatureF),
	}

	_, err := h.store.StopPlunge(r.Context(), params)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to stop the plunge timer", err)
		return
	}

	utils.RespondWithNoContent(w, http.StatusNoContent)
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
