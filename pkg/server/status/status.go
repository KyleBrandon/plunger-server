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
	"github.com/KyleBrandon/plunger-server/pkg/server/monitor"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func NewHandler(mctx *monitor.MonitorContext, store StatusStore, sensors sensor.Sensors, originPatterns []string) *Handler {
	h := Handler{
		mctx,
		store,
		sensors,
		PlungeState{},
		originPatterns,
	}

	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v1/status/ws", h.handleStatusWS)
}

func (h *Handler) handleStatusWS(w http.ResponseWriter, r *http.Request) {
	slog.Debug(">>handleWS: new incoming connection")
	defer slog.Debug("<<handleWS")

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

	h.monitorStatus(ctx, c)
}

func (h *Handler) monitorStatus(ctx context.Context, c *websocket.Conn) {
	slog.Debug(">>monitorStatus")
	defer slog.Debug("<<monitorStatus")

	ticker := time.NewTicker(1 * time.Second)
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			slog.Info("monitorStatus: client disconnected")
			c.Close(websocket.StatusNormalClosure, "Connection closed")
			return

		case <-ticker.C:

			// create a slice for any system messages
			errorMessages := make([]string, 0)

			h.mctx.Lock()
			roomTemp := h.mctx.RoomTemperature
			waterTemp := h.mctx.WaterTemperature

			h.mctx.Unlock()

			leakDetected, err := h.sensors.IsLeakPresent()
			if err != nil {
				errorMessages = append(errorMessages, err.Error())
			}

			pumpIsOn, err := h.sensors.IsPumpOn()
			if err != nil {
				errorMessages = append(errorMessages, err.Error())
			}

			ps, err := h.buildPlungeStatus(ctx, roomTemp, waterTemp)
			if err != nil {
				errorMessages = append(errorMessages, err.Error())
			}

			os, err := h.buildOzoneStatus(ctx)
			if err != nil {
				errorMessages = append(errorMessages, err.Error())
			}

			fs, err := h.buildFilterStatus(ctx)
			if err != nil {
				errorMessages = append(errorMessages, err.Error())
			}

			status := SystemStatus{
				ErrorMessages: errorMessages,
				PlungeStatus:  ps,
				OzoneStatus:   os,
				WaterTemp:     waterTemp,
				RoomTemp:      roomTemp,
				LeakDetected:  leakDetected,
				PumpOn:        pumpIsOn,
				FilterStatus:  fs,
			}

			err = wsjson.Write(ctx, c, status)
			if err != nil {
				slog.Error("monitorStatus: error writing to client", "error", err)
				c.Close(websocket.StatusInternalError, "error writing status")
				return
			}

		case <-heartbeatTicker.C:
			err := c.Ping(ctx)
			if err != nil {
				slog.Error("monitorStatus: error sending ping", "error", err)
				c.Close(websocket.StatusInternalError, "error sending ping")
				return
			}
		}
	}
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
	} else {
		// If it's not running, the reset the average temperature tracker
		h.state.MU.Lock()
		h.state.WaterTempTotal = 0.0
		h.state.RoomTempTotal = 0.0
		h.state.TempReadCount = 0
		h.state.MU.Unlock()
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

	fs := FilterStatus{
		ChangedAt: filter.ChangedAt,
		RemindAt:  filter.RemindAt,
		ChangeDue: time.Now().UTC().After(filter.RemindAt),
	}

	return fs, nil
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
