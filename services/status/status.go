package status

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func NewHandler(store StatusStore, sensors sensor.Sensors, originPatterns []string) *Handler {
	hub := NewHub()
	h := &Handler{
		store,
		sensors,
		PlungeState{},
		originPatterns,
		hub,
	}

	go h.run() // Start the hub's main loop

	return h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v2/status/ws", h.handleStatusWS)
}

func (h *Handler) handleStatusWS(w http.ResponseWriter, r *http.Request) {
	slog.Info(">>handleWS: new incoming connection")
	opts := &websocket.AcceptOptions{
		OriginPatterns: h.originPatterns,
	}
	c, err := websocket.Accept(w, r, opts)
	if err != nil {
		slog.Error("websocket accept error:", "error", err)
		return
	}

	defer c.Close(websocket.StatusInternalError, "Unexpected connection close")

	ctx := c.CloseRead(r.Context())
	client := &Client{
		conn: c,
		ctx:  ctx,
	}

	// register the client with the hub
	h.hub.register <- client

	// wait until the client disconnects or an error occurrs
	<-ctx.Done()

	// unregister the client from the hub
	h.hub.unregister <- client

	slog.Info("<<handleWS")
}

// NewHub creates and returns a new Hub instance
func NewHub() *Hub {
	hub := &Hub{
		clients:    make(map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
	return hub
}

// run manages registration, unregistration, and broadcasting
func (h *Handler) run() {
	ticker := time.NewTicker(1 * time.Second)
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer heartbeatTicker.Stop()

	hub := h.hub

	for {
		select {
		case client := <-hub.register:
			hub.mutex.Lock()
			hub.clients[client] = struct{}{}
			hub.mutex.Unlock()
			slog.Info("New client registered")

		case client := <-hub.unregister:
			hub.mutex.Lock()
			delete(hub.clients, client)
			hub.mutex.Unlock()
			slog.Info("Client unregistered")

		case <-ticker.C:
			h.broadcastStatus()

		case <-heartbeatTicker.C:
			for client := range hub.clients {
				err := client.conn.Ping(client.ctx)
				if err != nil {
					slog.Error("monitorPlunge: error sending ping", "error", err)
					hub.unregister <- client
				}

			}
		}
	}
}

// broadcastStatus sends the current system status to all connected clients
func (h *Handler) broadcastStatus() {
	h.hub.mutex.Lock()
	defer h.hub.mutex.Unlock()

	status := h.readStatus(context.Background())
	for client := range h.hub.clients {
		// Send the status to each client
		err := wsjson.Write(context.Background(), client.conn, status)
		if err != nil {
			slog.Error("Error writing to client", "error", err)
			client.conn.Close(websocket.StatusInternalError, "Error writing status")
			delete(h.hub.clients, client) // Remove the client on error
		}
	}
}

func (h *Handler) readStatus(ctx context.Context) SystemStatus {
	slog.Info(">>monitorPlunge")
	defer slog.Info("<<monitorPlunge")

	roomTemp, roomTempError, waterTemp, waterTempError := h.getRecentTemperatures(ctx)
	leakError := ""
	leakDetected, err := h.sensors.IsLeakPresent()
	if err != nil {
		leakError = err.Error()
	}

	pumpError := ""
	pumpIsOn, err := h.sensors.IsPumpOn()
	if err != nil {
		pumpError = err.Error()
	}

	plungeError := ""
	ps, err := h.buildPlungeStatus(ctx, roomTemp, waterTemp)
	if err != nil {
		plungeError = err.Error()
	}

	ozoneError := ""
	os, err := h.buildOzoneStatus(ctx)
	if err != nil {
		ozoneError = err.Error()
	}

	filterError := ""
	fs, err := h.buildFilterStatus(ctx)
	if err != nil {
		filterError = err.Error()
	}

	status := SystemStatus{
		PlungeStatus:   ps,
		PlungeError:    plungeError,
		OzoneStatus:    os,
		OzoneError:     ozoneError,
		WaterTemp:      waterTemp,
		WaterTempError: waterTempError,
		RoomTemp:       roomTemp,
		RoomTempError:  roomTempError,
		LeakDetected:   leakDetected,
		LeakError:      leakError,
		PumpOn:         pumpIsOn,
		PumpError:      pumpError,
		FilterStatus:   fs,
		FilterError:    filterError,
	}

	return status
}

func (h *Handler) buildPlungeStatus(ctx context.Context, roomTemp float64, waterTemp float64) (PlungeStatus, error) {
	p, err := h.store.GetLatestPlunge(ctx)
	if err != nil {
		// failed to read the plunge status.
		return PlungeStatus{}, err
	}

	var elapsedTime time.Duration
	if p.EndTime.Valid {
		elapsedTime = p.EndTime.Time.Sub(p.StartTime.Time)
	} else {
		elapsedTime = time.Since(p.StartTime.Time)
	}

	duration := time.Duration(p.ExpectedDuration) * time.Second
	remaining := duration - elapsedTime
	if remaining <= 0 {
		remaining = 0
	}

	avgWaterTemp, err := strconv.ParseFloat(p.AvgWaterTemp, 64)
	if err != nil {
		avgWaterTemp = 0.0
	}

	avgRoomTemp, err := strconv.ParseFloat(p.AvgRoomTemp, 64)
	if err != nil {
		avgRoomTemp = 0.0
	}

	// while the plunge is running, track live average temperatures, otherwise display what's stored in the database
	if p.Running {
		h.state.MU.Lock()
		avgRoomTemp, avgWaterTemp = h.updateAverageTemperatures(roomTemp, waterTemp)
		h.state.MU.Unlock()

		arg := database.UpdatePlungeAvgTempParams{
			ID:           p.ID,
			AvgRoomTemp:  fmt.Sprintf("%f", avgRoomTemp),
			AvgWaterTemp: fmt.Sprintf("%f", avgWaterTemp),
		}
		_, err = h.store.UpdatePlungeAvgTemp(ctx, arg)
		if err != nil {
			slog.Error("Failed to update current plunge avgerage temperature", "error", err)
		}
	}

	ps := PlungeStatus{
		StartTime:        p.StartTime.Time,
		StartWaterTemp:   p.StartWaterTemp,
		StartRoomTemp:    p.StartRoomTemp,
		EndTime:          p.EndTime.Time,
		EndWaterTemp:     p.EndWaterTemp,
		EndRoomTemp:      p.EndRoomTemp,
		Running:          p.Running,
		ExpectedDuration: int32(duration.Seconds()),
		Remaining:        remaining.Seconds(),
		ElapsedTime:      elapsedTime.Seconds(),
		AvgWaterTemp:     avgWaterTemp,
		AvgRoomTemp:      avgRoomTemp,
	}
	return ps, nil
}

func (h *Handler) buildOzoneStatus(ctx context.Context) (OzoneStatus, error) {
	ozone, err := h.store.GetLatestOzoneEntry(ctx)
	if err != nil {
		return OzoneStatus{}, err
	}

	var remaining time.Duration
	var endTime time.Time
	if ozone.Running {
		elapsedTime := time.Since(ozone.StartTime.Time)
		duration := time.Duration(ozone.ExpectedDuration) * time.Minute
		remaining = duration - elapsedTime
		endTime = ozone.StartTime.Time.Add(duration)
	} else {
		remaining = 0.0
		endTime = ozone.EndTime.Time
	}

	os := OzoneStatus{
		Running:     ozone.Running,
		Status:      ozone.StatusMessage.String,
		StartTime:   ozone.StartTime.Time,
		EndTime:     endTime,
		SecondsLeft: remaining.Seconds(),
	}

	return os, nil
}

func (h *Handler) buildFilterStatus(ctx context.Context) (FilterStatus, error) {
	filter, err := h.store.GetLatestFilterChange(ctx)
	if err != nil {
		return FilterStatus{}, err
	}

	if time.Now().UTC().After(filter.RemindAt) {
	}

	fs := FilterStatus{
		ChangedAt: filter.ChangedAt,
		RemindAt:  filter.RemindAt,
		ChangeDue: time.Now().UTC().After(filter.RemindAt),
	}

	return fs, nil
}

func (h *Handler) getRecentTemperatures(ctx context.Context) (float64, string, float64, string) {
	roomTemp := 0.0
	roomTempError := ""
	waterTemp := 0.0
	waterTempError := ""

	temperature, err := h.store.FindMostRecentTemperatures(ctx)
	// if the room temperature was read successfully, convert it into a float
	if err == nil && temperature.RoomTemp.Valid {
		roomTemp, err = strconv.ParseFloat(temperature.RoomTemp.String, 64)
	}

	// if we failed to read or conver the room temperature set it to a default and set an error message
	if err != nil || !temperature.RoomTemp.Valid {
		roomTemp = 0.0
		roomTempError = "No current room temperature"
	}

	// if the water temperature was read successfully, convert it into a float
	if err == nil && temperature.WaterTemp.Valid {
		waterTemp, err = strconv.ParseFloat(temperature.WaterTemp.String, 64)
	}

	// if we failed to read or conver the room temperature set it to a default and set an error message
	if err != nil || !temperature.WaterTemp.Valid {
		waterTemp = 0.0
		waterTempError = "No current water temperature"
	}

	return roomTemp, roomTempError, waterTemp, waterTempError
}

// Update the temperatures
func (h *Handler) updateAverageTemperatures(roomTemp float64, waterTemp float64) (float64, float64) {
	h.state.WaterTempTotal += waterTemp
	h.state.RoomTempTotal += roomTemp
	h.state.TempReadCount++
	avgWaterTemp := h.state.WaterTempTotal / float64(h.state.TempReadCount)
	avgRoomTemp := h.state.RoomTempTotal / float64(h.state.TempReadCount)

	return avgRoomTemp, avgWaterTemp
}
