package monitor

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
)

func NewHandler(store MonitorStore, sensors sensor.Sensors) *Handler {
	return &Handler{
		store,
		sensors,
	}
}

func (h *Handler) StartMonitorJobs(ctx context.Context, cancel context.CancelFunc) {
	go h.monitorTemperatures(ctx, cancel)
	go h.monitorOzone(ctx, cancel)
}

func (h *Handler) monitorTemperatures(ctx context.Context, cancel context.CancelFunc) {
	for {
		select {
		case <-ctx.Done():
			slog.Info("<<monitorTemperatures")
			return

		case <-time.After(30 * time.Second):

			waterTemp := sql.NullString{
				Valid: false,
			}
			roomTemp := sql.NullString{
				Valid: false,
			}

			rt, wt := h.sensors.ReadRoomAndWaterTemperature()
			if rt.Err == nil {
				roomTemp.Valid = true
				roomTemp.String = fmt.Sprintf("%f", rt.TemperatureF)
			} else {
				slog.Error("failed to read the room temperature", "error", rt.Err)
			}

			if wt.Err == nil {
				waterTemp.Valid = true
				waterTemp.String = fmt.Sprintf("%f", wt.TemperatureF)
			} else {
				slog.Error("failed to read the water temperature", "error", wt.Err)
			}

			arg := database.SaveTemperatureParams{
				WaterTemp: waterTemp,
				RoomTemp:  roomTemp,
			}
			_, err := h.store.SaveTemperature(ctx, arg)
			if err != nil {
				slog.Error("failed to save the room and water temperatures", "error", err)
			}
		}
	}
}

func (h *Handler) monitorOzone(ctx context.Context, cancel context.CancelFunc) {
	slog.Debug(">>monitorOzone")
	defer slog.Debug("<<monitorOzone")
	// start with the ozone off
	h.sensors.TurnOzoneOff()

	for {
		select {
		case <-ctx.Done():
			err := h.sensors.TurnOzoneOff()
			if err != nil {
				slog.Error("failed to turn ozone off when exiting the ozone monitor", "error", err)
				// TODO: Notify user in Status
			}

			return

		case <-time.After(5 * time.Second):

			ozone, err := h.store.GetLatestOzone(ctx)
			if err != nil {
				slog.Error("failed to query the latest ozone job", "error", err)
				continue
			}

			if ozone.Running {
				elapsedTime := time.Since(ozone.StartTime.Time)

				duration := time.Duration(ozone.ExpectedDuration) * time.Minute
				remaining := duration - elapsedTime
				if remaining <= 0 {
					h.sensors.TurnOzoneOff()
					h.store.StopOzone(ctx, ozone.ID)
				}
			}

		}
	}
}
