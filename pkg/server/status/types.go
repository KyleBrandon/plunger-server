package status

import (
	"context"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/pkg/server/monitor"
)

type (
	OzoneStatus struct {
		Running     bool      `json:"running"`
		StartTime   time.Time `json:"start_time,omitempty"`
		EndTime     time.Time `json:"end_time,omitempty"`
		SecondsLeft float64   `json:"seconds_left"`
	}

	PlungeStatus struct {
		StartTime      time.Time `json:"start_time,omitempty"`
		EndTime        time.Time `json:"end_time,omitempty"`
		StartWaterTemp float64   `json:"start_water_temp"`
		StartRoomTemp  float64   `json:"start_room_temp"`
		EndWaterTemp   float64   `json:"end_water_temp"`
		EndRoomTemp    float64   `json:"end_room_temp"`

		Running          bool    `json:"running"`
		ExpectedDuration int32   `json:"expected_duration"`
		Remaining        float64 `json:"remaining_time"`
		ElapsedTime      float64 `json:"elapsed_time"`
		AvgWaterTemp     float64 `json:"average_water_temp"`
		AvgRoomTemp      float64 `json:"average_room_temp"`
	}

	FilterStatus struct {
		ChangedAt time.Time `json:"changed_at,omitempty"`
		RemindAt  time.Time `json:"remind_at,omitempty"`
		ChangeDue bool      `json:"change_due"`
	}

	TemperatureStatus struct {
		WaterTemp             float64 `json:"water_temp"`
		RoomTemp              float64 `json:"room_temp"`
		MonitoringTemperature bool    `json:"monitor_target_temp"`
		TargetTemp            float64 `json:"target_temp"`
	}

	SystemStatus struct {
		AlertMessages     []string          `json:"alert_messages"`
		ErrorMessages     []string          `json:"error_messages"`
		LeakDetected      bool              `json:"leak_detected"`
		PumpOn            bool              `json:"pump_on"`
		TemperatureStatus TemperatureStatus `json:"temperature"`
		PlungeStatus      PlungeStatus      `json:"plunge"`
		OzoneStatus       OzoneStatus       `json:"ozone"`
		FilterStatus      FilterStatus      `json:"filter"`
	}

	StatusStore interface {
		FindMostRecentTemperatures(ctx context.Context) (database.Temperature, error)
		GetLatestPlunge(ctx context.Context) (database.Plunge, error)
		UpdatePlungeAvgTemp(ctx context.Context, arg database.UpdatePlungeAvgTempParams) (database.Plunge, error)
		GetLatestOzoneEntry(ctx context.Context) (database.Ozone, error)
		GetLatestFilterChange(ctx context.Context) (database.Filter, error)
	}

	Handler struct {
		mctx           *monitor.MonitorContext
		store          StatusStore
		sensors        sensor.Sensors
		originPatterns []string
	}
)
