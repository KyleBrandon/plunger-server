package status

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/services/plunges/v2"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func NewHandler(store plunges.PlungeStore, jobStore jobs.JobStore, sensors sensor.Sensors, state *plunges.PlungeState) *Handler {
	h := Handler{
		store,
		jobStore,
		sensors,
		state,
	}

	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v2/status/ws", h.handleStatusWS)
}

func (h *Handler) handleStatusWS(w http.ResponseWriter, r *http.Request) {
	slog.Info(">>handleWS: new incoming connection")
	// TODO: put these in a config
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

func (h *Handler) buildPlungeStatus(ctx context.Context) (PlungeStatus, error) {
	// get the most recent plunge and see if it's still running
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

	avgWaterTemp := 0.0
	waterTempError := ""
	avgRoomTemp := 0.0
	roomTempError := ""
	roomTemp, waterTemp := h.sensors.ReadRoomAndWaterTemperature()
	if roomTemp.Err != nil {
		roomTempError = roomTemp.Err.Error()
	}

	if waterTemp.Err != nil {
		waterTempError = waterTemp.Err.Error()
	}

	h.state.MU.Lock()
	avgRoomTemp, avgWaterTemp = h.state.UpdateAverageTemperatures(roomTemp.TemperatureF, waterTemp.TemperatureF)
	h.state.MU.Unlock()

	ps := PlungeStatus{
		ID:               p.ID,
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
		WaterTempError:   waterTempError,
		AvgRoomTemp:      avgRoomTemp,
		RoomTempError:    roomTempError,
	}
	return ps, nil
}

func (h *Handler) buildOzoneStatus(ctx context.Context) (OzoneStatus, error) {
	job, err := h.jobStore.GetLatestJobByType(ctx, jobs.JOBTYPE_OZONE_TIMER)
	if err != nil {
		return OzoneStatus{}, err
	}

	var status string
	var timeLeft float64
	if job.Status == jobs.JOBSTATUS_STARTED {
		status = "Running"
		timeLeft = job.EndTime.Sub(time.Now().UTC()).Seconds()
	} else {
		status = "Stopped"
		timeLeft = 0.0
	}

	os := OzoneStatus{
		ID:              job.ID,
		Status:          status,
		StartTime:       job.StartTime,
		EndTime:         job.EndTime,
		SecondsLeft:     timeLeft,
		Result:          job.Result.String,
		CancelRequested: job.CancelRequested,
	}

	return os, nil
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

			roomTemp, waterTemp := h.sensors.ReadRoomAndWaterTemperature()
			roomTempError := ""
			if roomTemp.Err != nil {
				roomTempError = roomTemp.Err.Error()
			}

			waterTempError := ""
			if waterTemp.Err != nil {
				waterTempError = waterTemp.Err.Error()
			}

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
			ps, err := h.buildPlungeStatus(ctx)
			if err != nil {
				plungeError = err.Error()
			}

			ozoneError := ""
			os, err := h.buildOzoneStatus(ctx)
			if err != nil {
				ozoneError = err.Error()
			}

			status := SystemStatus{
				PlungeStatus:   ps,
				PlungeError:    plungeError,
				OzoneStatus:    os,
				OzoneError:     ozoneError,
				WaterTemp:      waterTemp.TemperatureF,
				WaterTempError: waterTempError,
				RoomTemp:       roomTemp.TemperatureF,
				RoomTempError:  roomTempError,
				LeakDetected:   leakDetected,
				LeakError:      leakError,
				PumpOn:         pumpIsOn,
				PumpError:      pumpError,
			}

			err = wsjson.Write(ctx, c, status)
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
