package monitor

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
)

// InitializeMonitorSync will initialize a new MonitorSync struct.
func InitializeMonitorSync() *MonitorSync {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	msync := MonitorSync{
		wg:         &wg,
		ctx:        ctx,
		CancelFunc: cancel,
		OzoneCh:    make(chan OzoneTask),
	}

	return &msync
}

// NewHandler will create a new Handler struct to manage the go routes that monitor the state of the plunge.
func NewHandler(msync *MonitorSync, store MonitorStore, sensors sensor.Sensors) *Handler {
	return &Handler{
		store:   store,
		sensors: sensors,
	}
}

// StartMonitorRoutines will start up the go routines that monitor the plunge
func (h *Handler) StartMonitorRoutines(msync *MonitorSync) {
	msync.wg.Add(1)
	go h.monitorTemperatures(msync)

	msync.wg.Add(1)
	go h.monitorOzone(msync)

	msync.wg.Add(1)
	go h.monitorLeaks(msync)
}

func (h *Handler) monitorOzone(msync *MonitorSync) {
	slog.Info(">>monitorOzone")
	defer slog.Info("<<monitorOzone")

	// close the waitgroup when the routine exits
	defer msync.wg.Done()

	// start and stop with the ozone off
	h.sensors.TurnOzoneOff()
	defer h.sensors.TurnOzoneOff()

	slog.Info("monitorOzone: wait for action or done")
	for {
		select {
		case <-msync.ctx.Done():
			slog.Info("monitorOzone: context done")
			return

		case task, ok := <-msync.OzoneCh:
			if !ok {
				slog.Error("The ozone notification channel was closed")
				return
			}

			// process notification from ozone Handler
			switch task.Action {
			case OZONEACTION_START:
				slog.Info("OZONEACTION_START")
				h.startOzoneGenerator(msync, task.Duration)

			case OZONEACTION_STOP:
				// cancel the ozone generator
				slog.Info("OZONEACTION_STOP")
				msync.Lock()
				if msync.OzoneRunning {
					slog.Info("cancel ozone")
					msync.OzoneCancel()
				}
				msync.Unlock()
			}
		}
	}
}

func (h *Handler) startOzoneGenerator(msync *MonitorSync, duration int) error {
	slog.Info(">>startOzoneGenerator")
	defer slog.Info("<<startOzoneGenerator")

	msync.Lock()
	defer msync.Unlock()

	// is the ozone generator already running?
	if msync.OzoneRunning {
		// TODO: deal with this better
		slog.Error("ozone is already running")
		return errors.New("ozone already running")
	}

	startTime := sql.NullTime{
		Time:  time.Now().UTC(),
		Valid: true,
	}

	args := database.StartOzoneGeneratorParams{
		StartTime:        startTime,
		ExpectedDuration: int32(duration),
	}

	_, err := h.store.StartOzoneGenerator(msync.ctx, args)
	if err != nil {
		slog.Error("failed to update database with ozone start", "error", err)
		return err
	}

	err = h.sensors.TurnOzoneOn()
	if err != nil {
		h.setOzoneErrorMessage(msync.ctx, "failed to turn on ozone generator", err)

		return err
	}

	// create a context for the ozone goroutine with a hard timeout
	ozoneCtx, cancel := context.WithTimeout(msync.ctx, time.Duration(duration)*time.Minute)
	msync.OzoneCancel = cancel
	msync.OzoneRunning = true

	go func() {
		slog.Info("Enter goroutine to monitor ozone")
		defer slog.Info("Exit goroutine to monitor ozone")

		for range ozoneCtx.Done() {
			slog.Info("Ozone generator was stopped")
			h.stopOzoneGenerator(msync)
			return
		}
	}()

	return nil
}

func (h *Handler) stopOzoneGenerator(msync *MonitorSync) error {
	slog.Info(">>stopOzoneGenerator")
	defer slog.Info("<<stopOzoneGenerator")

	// turn ozone off no matter what
	err := h.sensors.TurnOzoneOff()
	if err != nil {
		// TODO: Notify user!!!
		h.setOzoneErrorMessage(msync.ctx, "failed to turn off ozone generator", err)
		return err
	}

	msync.Lock()
	msync.OzoneRunning = false
	msync.Unlock()

	ozone, err := h.store.GetLatestOzoneEntry(msync.ctx)
	if err != nil {
		slog.Error("failed to query database for latest ozone entry", "error", err)
		return err
	}

	// update the database and stop the ozone
	_, err = h.store.StopOzoneGenerator(msync.ctx, ozone.ID)
	if err != nil {
		slog.Error("failed to update the database with the ozone stop")
		return err
	}

	return nil
}

func (h *Handler) setOzoneErrorMessage(ctx context.Context, statusMessage string, err error) error {
	// TODO: Notify the user!!!
	//
	//

	slog.Error(statusMessage, "error", err)

	ozone, err := h.store.GetLatestOzoneEntry(ctx)
	if err != nil {
		slog.Error("failed to query database for latest ozone entry", "error", err)
		return err
	}

	// Update the ozone status to indicate it was not turned off
	arg := database.UpdateOzoneEntryStatusParams{
		ID:            ozone.ID,
		StatusMessage: sql.NullString{Valid: true, String: statusMessage},
	}

	_, dbErr := h.store.UpdateOzoneEntryStatus(ctx, arg)
	if dbErr != nil {
		// we log the database error but we don't return it, we want to know if we didn't stop the ozone generator
		slog.Error("failed to update ozone status to indicate ozone was not turned off", "error", err)
		return err
	}

	return nil
}

func (h *Handler) monitorTemperatures(msync *MonitorSync) {
	slog.Info(">>monitorTemperatures")
	defer slog.Info("<<monitorTemperatures")

	defer msync.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-msync.ctx.Done():
			slog.Info("monitorTemperatures: context done")
			return

		case <-ticker.C:

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
			_, err := h.store.SaveTemperature(msync.ctx, arg)
			if err != nil {
				slog.Error("failed to save the room and water temperatures", "error", err)
			}
		}
	}
}

func (h *Handler) monitorLeaks(msync *MonitorSync) {
	slog.Info(">>monitorLeaks")
	defer slog.Info("<<monitorLeaks")

	defer msync.wg.Done()

	// take an initial reading of the leak sensor so we can detect transitions from true/false
	prevLeakReading, err := h.sensors.IsLeakPresent()
	if err != nil {
		slog.Warn("failed to read sensor to determine if a leak is present", "error", err)
	}

	// if there is a leak present at start create a leak entry
	if prevLeakReading {
		_, err := h.store.CreateLeakDetected(msync.ctx, time.Now().UTC())
		if err != nil {
			slog.Error("failed to store leak detection in database", "error", err)
			// TODO: we should have alternative means of reporting this
		}
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {

		case <-msync.ctx.Done():
			// task was canceled or timedout
			slog.Info("monitorLeaks: context done")
			return

		case <-ticker.C:

			currentLeakReading, err := h.sensors.IsLeakPresent()
			if err != nil {
				slog.Warn("failed to read if leak was present", "error", err)
			}

			// have we had a change since we last read the sensor?
			if prevLeakReading != currentLeakReading {
				h.processLeakReading(msync.ctx, currentLeakReading)

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
		leak, err = h.store.GetLatestLeakDetected(ctx)
		if err != nil {
			slog.Warn("failed to read the latest leak from the database, create a new entry", "error", err)
			return err
		}

		// the entry's cleared_at should not be set
		if !leak.ClearedAt.Valid {
			leak, err = h.store.ClearDetectedLeak(ctx, leak.ID)
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

// CancelAndWait for the monitor sync routines to exit.
func (ms *MonitorSync) CancelAndWait() {
	// If the server stopped, cancel the monitor go routines
	ms.CancelFunc()

	// wait until all go routines have exited
	ms.wg.Wait()
}
