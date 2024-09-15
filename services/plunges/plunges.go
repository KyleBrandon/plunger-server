package plunges

import (
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/utils"
	"github.com/google/uuid"
	"nhooyr.io/websocket"
)

const DefaultPlungeDurationSeconds = "180"

func NewHandler(store PlungeStore, sensors Sensors) *Handler {
	h := Handler{}

	h.store = store
	h.sensors = sensors
	h.Running = false
	h.clients = make(map[*websocket.Conn]bool)
	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/plunge/status", h.handlePlungesGet)
	mux.HandleFunc("POST /v1/plunge/start", h.handlePlungesStart)
	mux.HandleFunc("PUT /v1/plunge/stop", h.handlePlungesStop)
}

func (h *Handler) handlePlungesGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesGet")
	h.plungeMu.Lock()
	defer h.plungeMu.Unlock()

	if !h.Running {
		utils.RespondWithError(w, http.StatusNotFound, "No active timer", nil)
		return
	}

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

	waterTemp, roomTemp, err := h.readCurrentTemperatures()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	// initalize the plunge context info
	h.plungeMu.Lock()
	defer h.plungeMu.Unlock()

	params := database.StartPlungeParams{
		StartTime:      sql.NullTime{Valid: true, Time: h.StartTime},
		StartWaterTemp: sql.NullString{Valid: true, String: fmt.Sprintf("%f", waterTemp)},
		StartRoomTemp:  sql.NullString{Valid: true, String: fmt.Sprintf("%f", roomTemp)},
	}

	plunge, err := h.store.StartPlunge(r.Context(), params)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	h.plungeID = plunge.ID
	h.StartTime = time.Now().UTC()
	h.Duration = time.Duration(duration) * time.Second
	h.Running = true
	h.ElapsedTime = 0

	go h.monitorPlunge()

	utils.RespondWithJSON(w, http.StatusCreated, databasePlungeToPlunge(plunge))
}

func (h *Handler) monitorPlunge() {
	for {
		h.plungeMu.Lock()
		remaining := h.Duration - time.Since(h.StartTime)
		if remaining <= 0 {
			remaining = 0
		}

		h.ElapsedTime = time.Since(h.StartTime)

		waterTemp, roomTemp, _ := h.readCurrentTemperatures()

		h.plungeMu.Unlock()

		status := PlungeStatus{
			Remaining: remaining,
			TotalTime: h.ElapsedTime,
			Running:   h.Running,
			WaterTemp: waterTemp,
			RoomTemp:  roomTemp,
		}

		h.broadcastToClients(status)

		if remaining == 0 && !h.Running {
			break
		}

		time.Sleep(1 * time.Second)
	}
}

func (h *Handler) handlePlungesStop(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesStop")

	h.plungeMu.Lock()
	defer h.plungeMu.Unlock()

	if !h.Running {
		utils.RespondWithError(w, http.StatusConflict, "No plunge timer running", nil)
		return
	}

	waterTemp, roomTemp, _ := h.readCurrentTemperatures()

	h.StopTime = time.Now().UTC()
	h.Running = false
	h.ElapsedTime = h.StopTime.Sub(h.StartTime)

	status := PlungeStatus{
		Remaining: 0,
		TotalTime: h.ElapsedTime,
		Running:   h.Running,
		WaterTemp: waterTemp,
		RoomTemp:  roomTemp,
	}

	h.broadcastToClients(status)

	params := database.StopPlungeParams{
		ID:           h.plungeID,
		EndTime:      sql.NullTime{Valid: true, Time: h.StopTime},
		EndWaterTemp: sql.NullString{Valid: true, String: fmt.Sprintf("%f", waterTemp)},
		EndRoomTemp:  sql.NullString{Valid: true, String: fmt.Sprintf("%f", roomTemp)},
	}

	_, err := h.store.StopPlunge(r.Context(), params)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	h.plungeID = uuid.Nil

	utils.RespondWithNoContent(w, http.StatusNoContent)
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// WebSocket handler
func (h *Handler) wsHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("Failed to upgrade WebSocket connection: %v\n", err)
		return
	}

	// Add the new client
	h.clientsMu.Lock()
	h.clients[conn] = true
	h.clientsMu.Unlock()

	// Handle incoming messages (optional, not needed for this timer example)
	go func() {
		defer conn.Close()
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				h.clientsMu.Lock()
				delete(h.clients, conn)
				h.clientsMu.Unlock()
				break
			}
		}
	}()
}

func (h *Handler) broadcastToClients(status PlungeStatus) {
	h.clientsMu.Lock()
	defer h.clientsMu.Unlock()

	for client := range h.clients {
		err := client.WriteJSON(message)
		if err != nil {
			fmt.Printf("Error writing to client: %v\n", err)
			client.Close()
			delete(h.clients, client)
		}
	}
}

func databasePlungeToPlunge(dbPlunge database.Plunge) PlungeResponse {
	resp := PlungeResponse{
		ID:        dbPlunge.ID,
		CreatedAt: dbPlunge.CreatedAt,
		UpdatedAt: dbPlunge.UpdatedAt,
	}

	if dbPlunge.StartTime.Valid {
		resp.StartTime = dbPlunge.StartTime.Time
	}
	if dbPlunge.EndTime.Valid {
		resp.EndTime = dbPlunge.EndTime.Time
		resp.ElapsedTime = resp.EndTime.Sub(resp.StartTime).Seconds()
		resp.Running = false
	} else {
		resp.ElapsedTime = time.Now().UTC().Sub(resp.StartTime).Seconds()
		resp.Running = true
	}
	if dbPlunge.StartWaterTemp.Valid {
		resp.StartWaterTemp = dbPlunge.StartWaterTemp.String
	}
	if dbPlunge.EndWaterTemp.Valid {
		resp.EndWaterTemp = dbPlunge.EndWaterTemp.String
	}
	if dbPlunge.StartRoomTemp.Valid {
		resp.StartRoomTemp = dbPlunge.StartRoomTemp.String
	}
	if dbPlunge.EndRoomTemp.Valid {
		resp.EndRoomTemp = dbPlunge.EndRoomTemp.String
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
		if temp.Name == "Room" {
			roomTemp = temp.TemperatureF
		} else if temp.Name == "Water" {
			waterTemp = temp.TemperatureF
		}
	}

	h.WaterTempTotal += waterTemp
	h.RoomTempTotal += roomTemp
	h.TempReadCount++

	return roomTemp, waterTemp, nil
}
