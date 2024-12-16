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
	OzoneAction int

	OzoneTask struct {
		Action   OzoneAction
		Duration int
	}

	NotificationTask struct {
		Message string
	}

	MonitorContext struct {
		sync.Mutex
		wg      *sync.WaitGroup
		ctx     context.Context
		store   MonitorStore
		sensors sensor.Sensors

		CancelFunc   context.CancelFunc
		OzoneRunning bool
		OzoneCh      chan OzoneTask
		OzoneCancel  context.CancelFunc
		NotifyCh     chan NotificationTask
		Notifier     *notify.Notify
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
