package plunges

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/utils"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

const DefaultPlungeDurationSeconds = "180"

func NewHandler(store PlungeStore, sensors Sensors) *Handler {
	h := Handler{}

	h.store = store
	h.sensors = sensors
	h.running = false
	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v2/plunges/ws", h.handleWS)
	mux.HandleFunc("GET /v2/plunges/status", h.handlePlungesGet)
	mux.HandleFunc("POST /v2/plunges/start", h.handlePlungesStart)
	mux.HandleFunc("PUT /v2/plunges/stop", h.handlePlungesStop)
}

func (h *Handler) handlePlungesGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesGet")
	h.mu.Lock()
	defer h.mu.Unlock()

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

	roomTemp, waterTemp, err := h.readCurrentTemperatures()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	params := database.StartPlungeParams{
		StartTime:        sql.NullTime{Valid: true, Time: time.Now().UTC()},
		StartWaterTemp:   fmt.Sprintf("%f", waterTemp),
		StartRoomTemp:    fmt.Sprintf("%f", roomTemp),
		ExpectedDuration: int32(duration),
	}

	// Save start to database
	plunge, err := h.store.StartPlunge(r.Context(), params)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	// initalize the plunge context info
	h.id = plunge.ID
	h.startTime = plunge.StartTime.Time
	h.duration = time.Duration(duration) * time.Second
	h.running = true

	utils.RespondWithJSON(w, http.StatusCreated, databasePlungeToPlunge(plunge))
}

func (h *Handler) handlePlungesStop(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesStop")

	h.mu.Lock()
	defer h.mu.Unlock()

	if !h.running {
		utils.RespondWithError(w, http.StatusConflict, "No plunge timer running", nil)
		return
	}

	roomTemp, waterTemp, _ := h.readCurrentTemperatures()
	avgWaterTemp := h.waterTempTotal / float64(h.tempReadCount)
	avgRoomTemp := h.roomTempTotal / float64(h.tempReadCount)

	h.stopTime = time.Now().UTC()
	h.running = false

	params := database.StopPlungeParams{
		ID:           h.id,
		EndTime:      sql.NullTime{Valid: true, Time: h.stopTime},
		EndWaterTemp: fmt.Sprintf("%f", waterTemp),
		EndRoomTemp:  fmt.Sprintf("%f", roomTemp),
		AvgWaterTemp: fmt.Sprintf("%f", avgWaterTemp),
		AvgRoomTemp:  fmt.Sprintf("%f", avgRoomTemp),
	}

	_, err := h.store.StopPlunge(r.Context(), params)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to stop the plunge timer", err)
		return
	}

	h.id = uuid.Nil

	utils.RespondWithNoContent(w, http.StatusNoContent)
}

func (h *Handler) handleWS(w http.ResponseWriter, r *http.Request) {
	slog.Info(">>handleWS: new incoming connection")
	opts := &websocket.AcceptOptions{
		OriginPatterns: []string{"localhost:3000", "http://localhost:3000", "10.0.4.213:3000", "http://10.0.4.213:3000"},
	}
	c, err := websocket.Accept(w, r, opts)
	if err != nil {
		slog.Error("websocket accept error:", "error", err)
		return
	}

	defer c.Close(websocket.StatusInternalError, "Unexpected connection close")

	ctx := c.CloseRead(r.Context())

	h.monitorPlunge(ctx, c)

	slog.Info("<<handleWS")
}

func (h *Handler) monitorPlunge(ctx context.Context, c *websocket.Conn) {
	slog.Info(">>monitorPlunge")
	ticker := time.NewTicker(1 * time.Second)
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("monitorPlunge: client disconnected")
			c.Close(websocket.StatusNormalClosure, "Connection closed")
			return

		case <-ticker.C:
			slog.Info("monitorPlunge: send status")
			h.mu.Lock()
			if !h.running {
				h.mu.Unlock()
				continue
			}

			elapsedTime := time.Since(h.startTime)
			remaining := h.duration - elapsedTime
			if remaining <= 0 {
				remaining = 0
			}

			roomTemp, waterTemp, _ := h.readCurrentTemperatures()
			avgWaterTemp := h.waterTempTotal / float64(h.tempReadCount)
			avgRoomTemp := h.roomTempTotal / float64(h.tempReadCount)

			h.mu.Unlock()

			status := PlungeStatus{
				ID:               h.id,
				ExpectedDuration: h.duration.Seconds(),
				Remaining:        remaining.Seconds(),
				ElapsedTime:      elapsedTime.Seconds(),
				Running:          h.running,
				WaterTemp:        waterTemp,
				RoomTemp:         roomTemp,
				AvgWaterTemp:     avgWaterTemp,
				AvgRoomTemp:      avgRoomTemp,
			}

			err := wsjson.Write(ctx, c, status)
			if err != nil {
				slog.Error("monitorPlunge: error writing to client", "error", err)
				c.Close(websocket.StatusInternalError, "error writing status")
				return
			}

		case <-heartbeatTicker.C:
			// send ping
			slog.Info("monitorPlunge: send ping")
			err := c.Ping(ctx)
			if err != nil {
				slog.Error("monitorPlunge: error sending ping", "error", err)
				c.Close(websocket.StatusInternalError, "error sending ping")
				return
			}
		}
	}
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

func (h *Handler) readCurrentTemperatures() (float64, float64, error) {
	temperatures, err := h.sensors.ReadTemperatures()
	if err != nil {
		return 0.0, 0.0, err
	}

	waterTemp := 0.0
	roomTemp := 0.0

	for _, temp := range temperatures {
		switch temp.Name {
		case "Room":
			roomTemp = temp.TemperatureF
		case "Water":
			waterTemp = temp.TemperatureF
		}
	}

	h.waterTempTotal += waterTemp
	h.roomTempTotal += roomTemp
	h.tempReadCount++

	return roomTemp, waterTemp, nil
}
