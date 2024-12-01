package monitor

import (
	"context"
	"sync"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/google/uuid"
)

const (
	OZONEACTION_START = 1
	OZONEACTION_STOP  = 2
)

type (
	OzoneAction int

	OzoneTask struct {
		Action   OzoneAction
		Duration time.Duration
	}

	MonitorSync struct {
		sync.Mutex
		wg         *sync.WaitGroup
		ctx        context.Context
		CancelFunc context.CancelFunc

		OzoneRunning bool
		OzoneCh      chan OzoneTask
		OzoneCancel  context.CancelFunc
	}

	Handler struct {
		store   MonitorStore
		sensors sensor.Sensors
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
