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
	}

	mctx.Ozone.OzoneCh = make(chan OzoneTask)
	mctx.Temperature.TempMonitorCh = make(chan TemperatureTask)
	mctx.Notification.NotifyCh = make(chan NotificationTask)
	mctx.Notification.notifier = notifier

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
	defer mctx.turnOzoneGeneratorOff()

	mctx.setInitialOzoneState()

	for {
		select {
		case <-mctx.ctx.Done():
			slog.Debug("monitorOzone: context done")
			return

		case task, ok := <-mctx.Ozone.OzoneCh:
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
				if mctx.Ozone.Running {
					slog.Debug("cancel ozone")
					mctx.Ozone.ozoneCancelFunc()
				}
				mctx.Unlock()
			}
		}
	}
}

func (mctx *MonitorContext) setInitialOzoneState() {
	mctx.Lock()
	defer mctx.Unlock()

	// make sure we start with the ozone off
	mctx.turnOzoneGeneratorOff()
	mctx.Ozone.Running = false

	// see if there was a last run
	ozone, err := mctx.store.GetLatestOzoneEntry(mctx.ctx)
	if err != nil {
		slog.Error("failed to query database for latest ozone entry", "error", err)
		return
	}

	// update ozone context with last ozone entry
	mctx.Ozone.Duration = int(ozone.ExpectedDuration)
	if ozone.StartTime.Valid {
		mctx.Ozone.StartTime = ozone.StartTime.Time
	}
	if ozone.EndTime.Valid {
		mctx.Ozone.EndTime = ozone.EndTime.Time
	}
}

// startOzoneGenerator will start the ozone generator for a specified duration in minutes
//
//	If the generator is already running, the current run will be stopped and a new run started.
func (mctx *MonitorContext) startOzoneGenerator(duration int) error {
	slog.Debug(">>startOzoneGenerator")
	defer slog.Debug("<<startOzoneGenerator")

	mctx.Lock()
	defer mctx.Unlock()

	// set the start time
	startTime := time.Now().UTC()

	// add a new database entry for starting the ozone generator
	err := mctx.startOzoneDatabase(startTime, duration)
	if err != nil {
		slog.Error("Failed to update database to start ozone", "error", err)
		return err
	}

	// start the ozone generator
	err = mctx.sensors.TurnOzoneOn()
	if err != nil {
		message := "Failed to turn the ozone generator on"
		slog.Error(message, "error", err)
		mctx.Notification.NotifyCh <- NotificationTask{Message: message}
		return err
	}

	// create a context for the ozone goroutine with a hard timeout
	ozoneCtx, cancel := context.WithTimeout(mctx.ctx, time.Duration(duration)*time.Minute)
	mctx.Ozone.ozoneCancelFunc = cancel
	mctx.Ozone.Running = true
	mctx.Ozone.Duration = duration
	mctx.Ozone.StartTime = startTime

	go func() {
		slog.Debug(">>monitor ozoneCancelFunc")
		defer slog.Debug("<<monitor ozoneCancelFunc")
		defer mctx.Ozone.ozoneCancelFunc()

		<-ozoneCtx.Done()
		mctx.stopOzoneGenerator()
	}()

	mctx.Notification.NotifyCh <- NotificationTask{Message: "Ozone generator was started"}

	return nil
}

func (mctx *MonitorContext) startOzoneDatabase(startTime time.Time, duration int) error {
	args := database.StartOzoneGeneratorParams{
		StartTime:        sql.NullTime{Time: startTime, Valid: true},
		ExpectedDuration: int32(duration),
	}

	_, err := mctx.store.StartOzoneGenerator(mctx.ctx, args)
	if err != nil {
		slog.Error("failed to update database with ozone start", "error", err)
		return err
	}
	return nil
}

func (mctx *MonitorContext) stopOzoneGenerator() error {
	slog.Debug(">>stopOzoneGenerator")
	defer slog.Debug("<<stopOzoneGenerator")

	// turn ozone off no matter what
	mctx.turnOzoneGeneratorOff()

	err := mctx.stopOzoneDatabase()
	if err != nil {
		slog.Error("failed to update database with ozone stop", "error", err)
	}

	mctx.Lock()
	mctx.Ozone.Running = false
	mctx.Ozone.EndTime = time.Now().UTC()
	mctx.Unlock()

	mctx.Notification.NotifyCh <- NotificationTask{Message: "Ozone generator was stopped"}

	return nil
}

func (mctx *MonitorContext) turnOzoneGeneratorOff() {
	err := mctx.sensors.TurnOzoneOff()
	if err != nil {
		message := "Failed to turn off ozone generator"
		slog.Error(message, "error", err)

		// ALWAYS notify the user that we were unable to turn the ozone generator off
		mctx.Notification.NotifyCh <- NotificationTask{Message: message}
	}
}

func (mctx *MonitorContext) stopOzoneDatabase() error {
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
			mctx.Temperature.WaterTemperature = wt.TemperatureF
			mctx.Temperature.RoomTemperature = rt.TemperatureF
			// are we monitoring for a target temperature?
			if mctx.Temperature.TemperatureMonitoring {
				lowRange := mctx.Temperature.TargetTemperature - 0.5
				highRange := mctx.Temperature.TargetTemperature + 0.5
				if wt.TemperatureF >= lowRange && wt.TemperatureF <= highRange {
					slog.Debug("Temperature in range", "target", mctx.Temperature.TargetTemperature, "lowRange", lowRange, "highRange", highRange)
					// notify  the user
					mctx.Notification.NotifyCh <- NotificationTask{Message: fmt.Sprintf("Target temperature %v was reached", mctx.Temperature.TargetTemperature)}
					mctx.Temperature.TemperatureMonitoring = false
				}
			}
			mctx.Unlock()

		case task, ok := <-mctx.Temperature.TempMonitorCh:
			if !ok {
				slog.Error("The temperature notification channel was closed")
				return
			}

			slog.Debug("Monitor temperature", "temperature", task.TargetTemperature)
			mctx.Lock()
			mctx.Temperature.TargetTemperature = task.TargetTemperature
			mctx.Temperature.TemperatureMonitoring = true
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
	slog.Debug(">>monitorNotifications")
	defer slog.Debug("<<monitorNotifications")

	defer mctx.wg.Done()
	for {
		select {
		case <-mctx.ctx.Done():
			slog.Debug("monitorNotifications: context done")
			return

		case task, ok := <-mctx.Notification.NotifyCh:
			if !ok {
				slog.Error("The notification channel was closed")
				return
			}

			// Send the SMS
			if mctx.Notification.notifier != nil {
				err := mctx.Notification.notifier.Send(
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
			slog.Warn("Inconsistent database state, we think there should be a leak that we are clearing")
		}
	}

	return nil
}
