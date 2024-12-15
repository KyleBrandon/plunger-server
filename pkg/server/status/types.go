package status

import (
	"context"
	"sync"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
)

type (
	OzoneStatus struct {
		Running     bool      `json:"running"`
		StartTime   time.Time `json:"start_time"`
		EndTime     time.Time `json:"end_time"`
		Status      string    `json:"status"`
		SecondsLeft float64   `json:"seconds_left"`
	}

	PlungeStatus struct {
		StartTime      time.Time `json:"start_time"`
		EndTime        time.Time `json:"end_time"`
		StartWaterTemp string    `json:"start_water_temp"`
		StartRoomTemp  string    `json:"start_room_temp"`
		EndWaterTemp   string    `json:"end_water_temp"`
		EndRoomTemp    string    `json:"end_room_temp"`

		Running          bool    `json:"running"`
		ExpectedDuration int32   `json:"expected_duration"`
		Remaining        float64 `json:"remaining_time"`
		ElapsedTime      float64 `json:"elapsed_time"`
		AvgWaterTemp     float64 `json:"average_water_temp"`
		AvgRoomTemp      float64 `json:"average_room_temp"`
	}

	FilterStatus struct {
		ChangedAt time.Time `json:"changed_at"`
		RemindAt  time.Time `json:"remind_at"`
		ChangeDue bool      `json:"change_due"`
	}

	SystemStatus struct {
		AlertMessages []string     `json:"alert_messages"`
		ErrorMessages []string     `json:"error_messages"`
		WaterTemp     float64      `json:"water_temp"`
		RoomTemp      float64      `json:"room_temp"`
		LeakDetected  bool         `json:"leak_detected"`
		PumpOn        bool         `json:"pump_on"`
		PlungeStatus  PlungeStatus `json:"plunge"`
		OzoneStatus   OzoneStatus  `json:"ozone"`
		FilterStatus  FilterStatus `json:"filter"`
	}

	PlungeState struct {
		MU             sync.Mutex
		WaterTempTotal float64
		RoomTempTotal  float64
		TempReadCount  int64
	}

	StatusStore interface {
		FindMostRecentTemperatures(ctx context.Context) (database.Temperature, error)
		GetLatestPlunge(ctx context.Context) (database.Plunge, error)
		UpdatePlungeAvgTemp(ctx context.Context, arg database.UpdatePlungeAvgTempParams) (database.Plunge, error)
		GetLatestOzoneEntry(ctx context.Context) (database.Ozone, error)
		GetLatestFilterChange(ctx context.Context) (database.Filter, error)
	}

	Handler struct {
		store          StatusStore
		sensors        sensor.Sensors
		state          PlungeState
		originPatterns []string
	}
)
