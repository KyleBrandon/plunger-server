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

func (h *Handler) StartMonitorJobs(ctx context.Context) {
	go h.monitorTemperatures(ctx)
	go h.monitorOzone(ctx)
}

func (h *Handler) monitorTemperatures(ctx context.Context) {
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

func (h *Handler) monitorOzone(ctx context.Context) {
	slog.Debug(">>monitorOzone")
	defer slog.Debug("<<monitorOzone")
	// start with the ozone off
	h.sensors.TurnOzoneOff()

	for {
		select {
		case <-ctx.Done():
			// if the ozone monitor is canceled then we should ensure the generator is stopped
			err := h.sensors.TurnOzoneOff()
			if err != nil {
				// this is about the best we can do currently since this happens when the server is shutting down
				slog.Error("failed to turn ozone off when exiting the ozone monitor", "error", err)
				// TODO: notify the user that we could not turn off the ozone generator when it was being shut down
			}

			return

		case <-time.After(5 * time.Second):

			ozone, err := h.store.GetLatestOzone(ctx)
			if err != nil {
				slog.Error("failed to query the latest ozone job", "error", err)
				continue
			}

			// while there is a recent ozone job that is running, check if it should be stopped
			if ozone.Running {
				// Ozone is running so determine if the duration has elapsed and turn off the generator if it has
				elapsedTime := time.Since(ozone.StartTime.Time)
				duration := time.Duration(ozone.ExpectedDuration) * time.Minute
				remaining := duration - elapsedTime
				if remaining <= 0 {
					// turn off the ozone generator
					err := h.sensors.TurnOzoneOff()
					if err != nil {
						slog.Error("Failed to turn off the ozone generator after the duration expired", "error", err)
						// TODO: notify the user that we could not turn off the ozone generator when it was done
					}

					// Update the databsae to indicate the ozone has stopped
					_, err = h.store.StopOzone(ctx, ozone.ID)
					if err != nil {
						slog.Error("Failed to update the databse to indicate the ozone job was finished", "error", err)
						// TODO: notify the user that we could not update the ozone job to indicate it was stopped
					}
				}
			} else {
				// safe guard to ensure that the ozone is stopped when it should be stopped
				err = h.sensors.TurnOzoneOff()
				if err != nil {
					slog.Warn("Failed to stop the ozone generator when the most recent job is stopped")
					// TODO: Notify the user that we can't turn off the ozone generator
				}
			}

		}
	}
}
