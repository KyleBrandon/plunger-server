package monitor

import (
	"context"
	"database/sql"
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
	slog.Debug(">>InitializeMonitorContext")
	defer slog.Debug("<<InitializeMonitorContext")

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	mctx := MonitorContext{
		wg:                &wg,
		ctx:               ctx,
		store:             store,
		sensors:           sensors,
		monitorCancelFunc: cancel,
		OzoneCh:           make(chan OzoneTask),
		NotifyCh:          make(chan NotificationTask),
		notifier:          notifier,
		TempMonitorCh:     make(chan TemperatureTask),
	}

	mctx.startMonitorRoutines()

	return &mctx
}

// CancelAndWait for the monitor sync routines to exit.
func (ms *MonitorContext) CancelAndWait() {
	// If the server stopped, cancel the monitor go routines
	ms.monitorCancelFunc()

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
					mctx.ozoneCancelFunc()
				}
				mctx.Unlock()
			}
		}
	}
}

func (mctx *MonitorContext) startOzoneGenerator(duration int) {
	slog.Debug(">>startOzoneGenerator")
	defer slog.Debug("<<startOzoneGenerator")

	mctx.Lock()
	defer mctx.Unlock()

	// is the ozone generator already running?
	if mctx.OzoneRunning {
		slog.Warn("ozone is already running")
		return
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
		return
	}

	err = mctx.sensors.TurnOzoneOn()
	if err != nil {
		mctx.setOzoneErrorMessage(mctx.ctx, "failed to turn on ozone generator", err)
		return
	}

	// create a context for the ozone goroutine with a hard timeout
	ozoneCtx, cancel := context.WithTimeout(mctx.ctx, time.Duration(duration)*time.Minute)
	mctx.ozoneCancelFunc = cancel
	mctx.OzoneRunning = true

	go func() {
		slog.Debug("Enter goroutine to monitor ozone")
		defer slog.Debug("Exit goroutine to monitor ozone")
		defer mctx.ozoneCancelFunc()

		<-ozoneCtx.Done()
		mctx.stopOzoneGenerator()
	}()

	mctx.NotifyCh <- NotificationTask{Message: "Ozone generator was started"}
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
			rt, wt := mctx.sensors.ReadRoomAndWaterTemperature()
			if rt.Err != nil {
				slog.Error("failed to read the room temperature", "error", rt.Err)
			}

			if wt.Err != nil {
				slog.Error("failed to read the water temperature", "error", wt.Err)
			}

			mctx.saveCurrentTemperatures(rt, wt)

			mctx.Lock()
			mctx.WaterTemperature = wt.TemperatureF
			mctx.RoomTemperature = rt.TemperatureF
			// are we monitoring for a target temperature?
			if mctx.temperatureMonitoring {
				lowRange := mctx.TargetTemperature - 0.5
				highRange := mctx.TargetTemperature + 0.5
				if wt.TemperatureF >= lowRange && wt.TemperatureF <= highRange {
					slog.Debug("Temperature in range", "target", mctx.TargetTemperature, "lowRange", lowRange, "highRange", highRange)
					// notify  the user
					mctx.NotifyCh <- NotificationTask{Message: fmt.Sprintf("Target temperature %v was reached", mctx.TargetTemperature)}
					mctx.temperatureMonitoring = false
				}

			}
			mctx.Unlock()

		case task, ok := <-mctx.TempMonitorCh:
			slog.Info("Received temperure task")
			if !ok {
				slog.Error("The temperature notification channel was closed")
				return
			}

			slog.Debug("Monitor temperature", "temperature", task.TargetTemperature)
			mctx.Lock()
			mctx.TargetTemperature = task.TargetTemperature
			mctx.temperatureMonitoring = true
			mctx.Unlock()
		}
	}
}

func (mctx *MonitorContext) saveCurrentTemperatures(rt sensor.TemperatureReading, wt sensor.TemperatureReading) {
	waterTemp := sql.NullString{
		Valid: false,
	}
	roomTemp := sql.NullString{
		Valid: false,
	}

	if rt.Err == nil {
		roomTemp.Valid = true
		roomTemp.String = fmt.Sprintf("%f", rt.TemperatureF)
	}

	if wt.Err == nil {
		waterTemp.Valid = true
		waterTemp.String = fmt.Sprintf("%f", wt.TemperatureF)
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

func (mctx *MonitorContext) monitorLeaks() {
	slog.Debug(">>monitorLeaks")
	defer slog.Debug("<<monitorLeaks")

	defer mctx.wg.Done()

	// take an initial reading of the leak sensor so we can detect transitions from true/false
	prevLeakReading, err := mctx.sensors.IsLeakPresent()
	if err != nil {
		slog.Warn("failed to read sensor to determine if a leak is present", "error", err)
	}

	notifyLeakDetected := true

	// if there is a leak present at start create a leak entry
	if prevLeakReading {
		_, err := mctx.store.CreateLeakDetected(mctx.ctx, time.Now().UTC())
		if err != nil {
			slog.Error("failed to mctx.store leak detection in database", "error", err)
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
			if currentLeakReading {
				if notifyLeakDetected {
					mctx.NotifyCh <- NotificationTask{Message: "Leak detected!! Turning off pump."}
					notifyLeakDetected = false
				}

				err = mctx.sensors.TurnPumpOff()
				if err != nil {
					slog.Error("failed to turn pump off while leak detected", "error", err)
					mctx.NotifyCh <- NotificationTask{Message: "Leak detected!! Failed to turn off pump."}
				}
			} else {
				// make sure to notify if a leak is detected
				notifyLeakDetected = true
			}
		}
	}
}

func (mctx *MonitorContext) monitorNotifications() {
	slog.Debug(">>monitorNotifications")
	defer slog.Debug("<<monitorNotifications")

	defer mctx.wg.Done()
	for {
		select {
		case <-mctx.ctx.Done():
			slog.Debug("monitorNotifications: context done")
			return

		case task, ok := <-mctx.NotifyCh:
			if !ok {
				slog.Error("The notification channel was closed")
				return
			}

			// Send the SMS
			if mctx.notifier != nil {
				err := mctx.notifier.Send(
					context.Background(),
					"Plunger Notification",
					task.Message,
				)
				if err != nil {
					slog.Error("failed to send message", "error", err, "message", task.Message)
				}
			} else {
				slog.Warn("Notifier is not registered for notifications")
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
