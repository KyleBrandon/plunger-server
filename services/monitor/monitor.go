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
	go h.monitorLeaks(ctx)
}

func (h *Handler) monitorTemperatures(ctx context.Context) {
	slog.Info(">>monitorTemperatures")
	defer slog.Info("<<monitorTemperatures")

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
	slog.Info(">>monitorOzone")
	defer slog.Info("<<monitorOzone")
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

				// if the remaining time is zero (ozone finished) then turn the ozone generator off
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

func (h *Handler) processLeakReading(ctx context.Context, leakDetected bool) error {
	var leak database.Leak
	var err error

	// if a leak was detected then create a new record to track it
	if leakDetected {
		leak, err = h.store.CreateLeakDetected(ctx, time.Now().UTC())
		if err != nil {
			slog.Error("failed to store leak detection in database", "error", err)
			// TODO: we should have alternative means of reporting this
		}
	} else {
		// if there is currently no leak, see if we need to report it being cleared
		leak, err = h.store.GetLatestLeak(ctx)
		if err != nil {
			slog.Warn("failed to read the latest leak from the database, create a new entry", "error", err)
			return err
		}

		// the entry's cleared_at should not be set
		if !leak.ClearedAt.Valid {
			leak, err = h.store.UpdateLeakCleared(ctx, leak.ID)
			if err != nil {
				slog.Error("failed to clear detected leak in database", "error", err)
			}
		} else {
			// we think there should be a leak that was cleared but the database already has a cleared
			slog.Warn("inconsistent database state, we think there should be a leak that we are clearing")
		}
	}

	return nil
}

func (h *Handler) monitorLeaks(ctx context.Context) {
	slog.Info(">>monitorLeaks")
	defer slog.Info("<<monitorLeaks")
	// take an initial reading of the leak sensor so we can detect transitions from true/false
	prevLeakReading, err := h.sensors.IsLeakPresent()
	if err != nil {
		slog.Warn("failed to read sensor to determine if a leak is present", "error", err)
	}

	// process the initial lead reading, this will create a leak record if one is detected
	h.processLeakReading(ctx, prevLeakReading)

	for {
		select {

		case <-ctx.Done():
			// task was canceled or timedout
			// config.StopJob(jobs.JOBTYPE_LEAK_MONITOR, "Success")

			return

		case <-time.After(5 * time.Second):

			currentLeakReading, err := h.sensors.IsLeakPresent()
			if err != nil {
				slog.Warn("failed to read if leak was present", "error", err)
			}

			// have we had a change since we last read the sensor?
			if prevLeakReading != currentLeakReading {
				h.processLeakReading(ctx, currentLeakReading)

				prevLeakReading = currentLeakReading
			}

			// if a leak was detected, then turn the pump off
			// TODO: this should be an event that we have listeners on
			if currentLeakReading {
				err = h.sensors.TurnPumpOff()
				if err != nil {
					// TODO: notify the user that we could not turn the pump off
					slog.Error("failed to turn pump off while leak detected", "error", err)
				}
			}
		}
	}
}
