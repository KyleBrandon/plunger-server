package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/google/uuid"
	"github.com/nikoksr/notify"
)

const (
	OZONEACTION_START = 1
	OZONEACTION_STOP  = 2
)

type (
	// OzoneAction indicates if the ozone generator should start or stop.
	//  Values can be:
	//      OZONEACTION_START
	//      OZONEACTION_STOP
	OzoneAction int

	// OzoneTask is a struct used to contain the information needed to run the ozone generator for a set duration.
	OzoneTask struct {
		Action OzoneAction

		// Duration to run the ozone generator in minutes.
		Duration int
	}

	// NotificationTask is a struct used to send messages to a destination.
	NotificationTask struct {
		// Message to send to the consumer.
		Message string
	}

	// TemperatureTask will start monitoring the temperature indicated.
	// If another TemperatureTask is received while already monitoring, it will replace the current monitor.
	// Once the temperature has been reached the user will be notified once.
	TemperatureTask struct {
		TargetTemperature float64
	}

	MonitorContext struct {
		sync.Mutex
		wg      *sync.WaitGroup
		ctx     context.Context
		store   MonitorStore
		sensors sensor.Sensors

		monitorCancelFunc context.CancelFunc

		OzoneCh         chan OzoneTask // OzoneCh is a channel that receives an OzoneTask to start or stop the ozone generator.
		ozoneCancelFunc context.CancelFunc
		ozoneRunning    bool

		NotifyCh chan NotificationTask // Channel to track notification tasks
		notifier *notify.Notify

		TempMonitorCh         chan TemperatureTask // Channel to track temperature monitoring requests
		targetTemperature     float64
		temperatureMonitoring bool
	}

	MonitorStore interface {
		SaveTemperature(ctx context.Context, arg database.SaveTemperatureParams) (database.Temperature, error)
		GetLatestOzoneEntry(ctx context.Context) (database.Ozone, error)
		StartOzoneGenerator(ctx context.Context, arg database.StartOzoneGeneratorParams) (database.Ozone, error)
		StopOzoneGenerator(ctx context.Context, id uuid.UUID) (database.Ozone, error)
		UpdateOzoneEntryStatus(ctx context.Context, args database.UpdateOzoneEntryStatusParams) (database.Ozone, error)
		GetLatestLeakDetected(ctx context.Context) (database.Leak, error)
		CreateLeakDetected(ctx context.Context, detectedAt time.Time) (database.Leak, error)
		ClearDetectedLeak(ctx context.Context, id uuid.UUID) (database.Leak, error)
	}
)
