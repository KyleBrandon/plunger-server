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
	"github.com/nikoksr/notify"
)

// InitializeMonitorContext will initialize a new MonitorSync struct.
func InitializeMonitorContext(notifier *notify.Notify, store MonitorStore, sensors sensor.Sensors) *MonitorContext {
	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	mctx := MonitorContext{
		wg:         &wg,
		ctx:        ctx,
		store:      store,
		sensors:    sensors,
		CancelFunc: cancel,
		OzoneCh:    make(chan OzoneTask),
		NotifyCh:   make(chan NotificationTask),
		Notifier:   notifier,
	}

	mctx.startMonitorRoutines()

	return &mctx
}

// CancelAndWait for the monitor sync routines to exit.
func (ms *MonitorContext) CancelAndWait() {
	// If the server stopped, cancel the monitor go routines
	ms.CancelFunc()

	// wait until all go routines have exited
	ms.wg.Wait()
}

// StartMonitorRoutines will start up the go routines that monitor the plunge
func (mctx *MonitorContext) startMonitorRoutines() {
	mctx.wg.Add(1)
	go mctx.monitorNotifications()

	mctx.wg.Add(1)
	go mctx.monitorTemperatures()

	mctx.wg.Add(1)
	go mctx.monitorOzone()

	mctx.wg.Add(1)
	go mctx.monitorLeaks()
}

func (mctx *MonitorContext) monitorOzone() {
	slog.Debug(">>monitorOzone")
	defer slog.Debug("<<monitorOzone")

	// close the waitgroup when the routine exits
	defer mctx.wg.Done()

	// start and stop with the ozone off
	mctx.sensors.TurnOzoneOff()
	defer mctx.sensors.TurnOzoneOff()

	for {
		select {
		case <-mctx.ctx.Done():
			slog.Debug("monitorOzone: context done")
			return

		case task, ok := <-mctx.OzoneCh:
			if !ok {
				slog.Error("The ozone notification channel was closed")
				return
			}

			// process notification from ozone Handler
			switch task.Action {
			case OZONEACTION_START:
				slog.Debug("OZONEACTION_START")
				mctx.startOzoneGenerator(task.Duration)

			case OZONEACTION_STOP:
				// cancel the ozone generator
				slog.Debug("OZONEACTION_STOP")
				mctx.Lock()
				if mctx.OzoneRunning {
					slog.Debug("cancel ozone")
					mctx.OzoneCancel()
				}
				mctx.Unlock()
			}
		}
	}
}

func (mctx *MonitorContext) startOzoneGenerator(duration int) error {
	slog.Debug(">>startOzoneGenerator")
	defer slog.Debug("<<startOzoneGenerator")

	mctx.Lock()
	defer mctx.Unlock()

	// is the ozone generator already running?
	if mctx.OzoneRunning {
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

	_, err := mctx.store.StartOzoneGenerator(mctx.ctx, args)
	if err != nil {
		slog.Error("failed to update database with ozone start", "error", err)
		return err
	}

	err = mctx.sensors.TurnOzoneOn()
	if err != nil {
		mctx.setOzoneErrorMessage(mctx.ctx, "failed to turn on ozone generator", err)

		return err
	}

	// create a context for the ozone goroutine with a hard timeout
	ozoneCtx, cancel := context.WithTimeout(mctx.ctx, time.Duration(duration)*time.Minute)
	mctx.OzoneCancel = cancel
	mctx.OzoneRunning = true

	go func() {
		slog.Debug("Enter goroutine to monitor ozone")
		defer slog.Debug("Exit goroutine to monitor ozone")

		<-ozoneCtx.Done()
		slog.Debug("Ozone generator was stopped")
		mctx.stopOzoneGenerator()
	}()

	mctx.NotifyCh <- NotificationTask{Message: "Ozone generator was started"}

	return nil
}

func (mctx *MonitorContext) stopOzoneGenerator() error {
	slog.Debug(">>stopOzoneGenerator")
	defer slog.Debug("<<stopOzoneGenerator")

	// turn ozone off no matter what
	err := mctx.sensors.TurnOzoneOff()
	if err != nil {
		mctx.setOzoneErrorMessage(mctx.ctx, "failed to turn off ozone generator", err)
		return err
	}

	mctx.Lock()
	mctx.OzoneRunning = false
	mctx.Unlock()

	ozone, err := mctx.store.GetLatestOzoneEntry(mctx.ctx)
	if err != nil {
		slog.Error("failed to query database for latest ozone entry", "error", err)
		return err
	}

	// update the database and stop the ozone
	_, err = mctx.store.StopOzoneGenerator(mctx.ctx, ozone.ID)
	if err != nil {
		slog.Error("failed to update the database with the ozone stop")
		return err
	}

	mctx.NotifyCh <- NotificationTask{Message: "Ozone generator was stopped"}

	return nil
}

func (mctx *MonitorContext) setOzoneErrorMessage(ctx context.Context, statusMessage string, err error) error {
	slog.Error(statusMessage, "error", err)

	// TODO: Should we have a message per ozone entry or something more general?
	ozone, err := mctx.store.GetLatestOzoneEntry(ctx)
	if err != nil {
		slog.Error("failed to query database for latest ozone entry", "error", err)
		return err
	}

	// Update the ozone status to indicate it was not turned off
	arg := database.UpdateOzoneEntryStatusParams{
		ID:            ozone.ID,
		StatusMessage: sql.NullString{Valid: true, String: statusMessage},
	}

	_, dbErr := mctx.store.UpdateOzoneEntryStatus(ctx, arg)
	if dbErr != nil {
		// we log the database error but we don't return it, we want to know if we didn't stop the ozone generator
		slog.Error("failed to update ozone status to indicate ozone was not turned off", "error", err)
		return err
	}

	mctx.NotifyCh <- NotificationTask{Message: statusMessage}

	return nil
}

func (mctx *MonitorContext) monitorTemperatures() {
	slog.Debug(">>monitorTemperatures")
	defer slog.Debug("<<monitorTemperatures")

	defer mctx.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-mctx.ctx.Done():
			slog.Debug("monitorTemperatures: context done")
			return

		case <-ticker.C:

			waterTemp := sql.NullString{
				Valid: false,
			}
			roomTemp := sql.NullString{
				Valid: false,
			}

			rt, wt := mctx.sensors.ReadRoomAndWaterTemperature()
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
			_, err := mctx.store.SaveTemperature(mctx.ctx, arg)
			if err != nil {
				slog.Error("failed to save the room and water temperatures", "error", err)
			}
		}
	}
}

func (mctx *MonitorContext) monitorLeaks() {
	slog.Debug(">>monitorLeaks")
	defer slog.Debug("<<monitorLeaks")

	defer mctx.wg.Done()

	// take an initial reading of the leak sensor so we can detect transitions from true/false
	prevLeakReading, err := mctx.sensors.IsLeakPresent()
	if err != nil {
		slog.Warn("failed to read sensor to determine if a leak is present", "error", err)
	}

	// if there is a leak present at start create a leak entry
	if prevLeakReading {
		_, err := mctx.store.CreateLeakDetected(mctx.ctx, time.Now().UTC())
		if err != nil {
			slog.Error("failed to mctx.store leak detection in database", "error", err)
			// TODO: we should have alternative means of reporting this
		}
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {

		case <-mctx.ctx.Done():
			// task was canceled or timedout
			slog.Debug("monitorLeaks: context done")
			return

		case <-ticker.C:

			currentLeakReading, err := mctx.sensors.IsLeakPresent()
			if err != nil {
				slog.Warn("failed to read if leak was present", "error", err)
			}

			// have we had a change since we last read the sensor?
			if prevLeakReading != currentLeakReading {
				mctx.processLeakReading(mctx.ctx, currentLeakReading)

				prevLeakReading = currentLeakReading
			}

			// if a leak was detected, then turn the pump off
			// TODO: this should be an event that we have listeners on
			if currentLeakReading {
				err = mctx.sensors.TurnPumpOff()
				if err != nil {
					// TODO: notify the user that we could not turn the pump off
					slog.Error("failed to turn pump off while leak detected", "error", err)
				}
			}
		}
	}
}

func (mctx *MonitorContext) monitorNotifications() {
	slog.Info(">>monitorNotifications")
	defer slog.Info("<<monitorNotifications")

	defer mctx.wg.Done()
	for {
		select {
		case <-mctx.ctx.Done():
			slog.Info("monitorNotifications: context done")
			return

		case task, ok := <-mctx.NotifyCh:
			if !ok {
				slog.Error("The notification channel was closed")
				return
			}

			slog.Info(task.Message)

			// Send the SMS
			err := mctx.Notifier.Send(
				context.Background(),
				"Plunger Notification",
				task.Message,
			)
			if err != nil {
				slog.Error("failed to send message", "error", err, "message", task.Message)
			}

		}
	}
}

func (mctx *MonitorContext) processLeakReading(ctx context.Context, leakDetected bool) error {
	var leak database.Leak
	var err error

	// if a leak was detected then create a new record to track it
	if leakDetected {
		leak, err = mctx.store.CreateLeakDetected(ctx, time.Now().UTC())
		if err != nil {
			slog.Error("failed to mctx.store leak detection in database", "error", err)
			// TODO: we should have alternative means of reporting this
		}
	} else {
		// if there is currently no leak, see if we need to report it being cleared
		leak, err = mctx.store.GetLatestLeakDetected(mctx.ctx)
		if err != nil {
			slog.Warn("failed to read the latest leak from the database, create a new entry", "error", err)
			return err
		}

		// the entry's cleared_at should not be set
		if !leak.ClearedAt.Valid {
			leak, err = mctx.store.ClearDetectedLeak(ctx, leak.ID)
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
