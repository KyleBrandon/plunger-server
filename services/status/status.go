package status

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/services/plunges/v2"
	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/google/uuid"
)

func NewHandler(store plunges.PlungeStore, jobStore jobs.JobStore, sensors sensor.Sensors, originPatterns []string) *Handler {
	h := Handler{
		store,
		jobStore,
		sensors,
		PlungeState{},
		originPatterns,
	}

	return &h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/v2/status/ws", h.handleStatusWS)
}

// Update the temperatures
func (h *Handler) updateAverageTemperatures(roomTemperature float64, waterTemperature float64) (float64, float64) {
	h.state.WaterTempTotal += waterTemperature
	h.state.RoomTempTotal += roomTemperature
	h.state.TempReadCount++
	avgWaterTemp := h.state.WaterTempTotal / float64(h.state.TempReadCount)
	avgRoomTemp := h.state.RoomTempTotal / float64(h.state.TempReadCount)

	return avgRoomTemp, avgWaterTemp
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

	h.monitorPlunge(ctx, c)

	slog.Info("<<handleWS")
}

func (h *Handler) buildPlungeStatus(ctx context.Context, wg *sync.WaitGroup) (PlungeStatus, error) {
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

	waterTempError := ""
	roomTempError := ""
	avgWaterTemp, err := strconv.ParseFloat(p.AvgWaterTemp, 64)
	if err != nil {
		waterTempError = err.Error()
	}

	avgRoomTemp, err := strconv.ParseFloat(p.AvgRoomTemp, 64)
	if err != nil {
		roomTempError = err.Error()
	}

	roomTemp, waterTemp := h.sensors.ReadRoomAndWaterTemperature()
	if roomTemp.Err != nil {
		roomTempError = roomTemp.Err.Error()
	}

	if waterTemp.Err != nil {
		waterTempError = waterTemp.Err.Error()
	}

	// while the plunge is running, track live average temperatures, otherwise display what's stored in the database
	if p.Running {
		h.state.MU.Lock()
		avgRoomTemp, avgWaterTemp = h.updateAverageTemperatures(roomTemp.TemperatureF, waterTemp.TemperatureF)
		h.state.MU.Unlock()

		wg.Add(1)
		go func(id uuid.UUID, avgRoomTemp float64, avgWaterTemp float64) {
			defer wg.Done()
			arg := database.UpdatePlungeAvgTempParams{
				ID:           p.ID,
				AvgRoomTemp:  fmt.Sprintf("%f", avgRoomTemp),
				AvgWaterTemp: fmt.Sprintf("%f", avgWaterTemp),
			}
			_, err = h.store.UpdatePlungeAvgTemp(ctx, arg)
			if err != nil {
				slog.Error("Failed to update current plunge avgerage temperature", "error", err)
			}
		}(p.ID, avgRoomTemp, avgWaterTemp)
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
	defer slog.Info("<<monitorPlunge")

	ticker := time.NewTicker(1 * time.Second)
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	defer heartbeatTicker.Stop()

	var wg sync.WaitGroup
	defer wg.Wait()

	for {
		select {
		case <-ctx.Done():
			slog.Info("monitorPlunge: client disconnected")
			c.Close(websocket.StatusNormalClosure, "Connection closed")
			return

		case <-ticker.C:

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
			ps, err := h.buildPlungeStatus(ctx, &wg)
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
			err := c.Ping(ctx)
			if err != nil {
				slog.Error("monitorPlunge: error sending ping", "error", err)
				c.Close(websocket.StatusInternalError, "error sending ping")
				return
			}
		}
	}
}
