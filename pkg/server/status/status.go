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
			roomTemp := h.mctx.Temperature.RoomTemperature
			waterTemp := h.mctx.Temperature.WaterTemperature

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

			os, err := h.buildOzoneStatus()
			if err != nil {
				errorMessages = append(errorMessages, err.Error())
			}

			fs, err := h.buildFilterStatus(ctx)
			if err != nil {
				errorMessages = append(errorMessages, err.Error())
			}

			ts, err := h.buildTemperatureStatus(roomTemp, waterTemp)
			if err != nil {
				errorMessages = append(errorMessages, err.Error())
			}

			status := SystemStatus{
				ErrorMessages:     errorMessages,
				TemperatureStatus: ts,
				PlungeStatus:      ps,
				OzoneStatus:       os,
				LeakDetected:      leakDetected,
				PumpOn:            pumpIsOn,
				FilterStatus:      fs,
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

func (h *Handler) buildTemperatureStatus(roomTemp float64, waterTemp float64) (TemperatureStatus, error) {
	h.mctx.Lock()
	defer h.mctx.Unlock()
	ts := TemperatureStatus{
		WaterTemp:             waterTemp,
		RoomTemp:              roomTemp,
		MonitoringTemperature: h.mctx.Temperature.TemperatureMonitoring,
		TargetTemp:            h.mctx.Temperature.TargetTemperature,
	}

	return ts, nil
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
		avgRoomTemp, avgWaterTemp = h.updateAverageTemperatures(roomTemp, waterTemp)

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
		h.mctx.Plunge.Lock()
		h.mctx.Plunge.WaterTempTotal = 0.0
		h.mctx.Plunge.RoomTempTotal = 0.0
		h.mctx.Plunge.TempReadCount = 0
		h.mctx.Plunge.Unlock()
	}

	startWaterTemp, err := strconv.ParseFloat(p.StartWaterTemp, 64)
	startRoomTemp, err := strconv.ParseFloat(p.StartRoomTemp, 64)
	endWaterTemp, err := strconv.ParseFloat(p.EndWaterTemp, 64)
	endRoomTemp, err := strconv.ParseFloat(p.EndRoomTemp, 64)

	ps := PlungeStatus{
		StartTime:        p.StartTime.Time,
		StartWaterTemp:   startWaterTemp,
		StartRoomTemp:    startRoomTemp,
		EndTime:          p.EndTime.Time,
		EndWaterTemp:     endWaterTemp,
		EndRoomTemp:      endRoomTemp,
		Running:          p.Running,
		ExpectedDuration: int32(duration.Seconds()),
		Remaining:        remaining.Seconds(),
		ElapsedTime:      elapsedTime.Seconds(),
		AvgWaterTemp:     avgWaterTemp,
		AvgRoomTemp:      avgRoomTemp,
	}
	return ps, nil
}

func (h *Handler) buildOzoneStatus() (OzoneStatus, error) {
	ozone := &h.mctx.Ozone

	var remaining time.Duration
	var endTime time.Time
	if h.mctx.Ozone.Running {
		elapsedTime := time.Since(ozone.StartTime)
		duration := time.Duration(ozone.Duration) * time.Minute
		remaining = duration - elapsedTime
		endTime = ozone.StartTime.Add(duration)
	} else {
		remaining = 0.0
		endTime = ozone.EndTime
	}

	os := OzoneStatus{
		Running:     ozone.Running,
		StartTime:   ozone.StartTime,
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
	h.mctx.Plunge.Lock()
	defer h.mctx.Plunge.Unlock()
	h.mctx.Plunge.WaterTempTotal += waterTemp
	h.mctx.Plunge.RoomTempTotal += roomTemp
	h.mctx.Plunge.TempReadCount++
	avgWaterTemp := h.mctx.Plunge.WaterTempTotal / float64(h.mctx.Plunge.TempReadCount)
	avgRoomTemp := h.mctx.Plunge.RoomTempTotal / float64(h.mctx.Plunge.TempReadCount)

	return avgRoomTemp, avgWaterTemp
}
