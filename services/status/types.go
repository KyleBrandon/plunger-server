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
		StartTime       time.Time `json:"start_time"`
		EndTime         time.Time `json:"end_time"`
		Status          string    `json:"status"`
		Result          string    `json:"result"`
		SecondsLeft     float64   `json:"seconds_left"`
		CancelRequested bool      `json:"cancel_requested"`
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

	SystemStatus struct {
		WaterTemp      float64      `json:"water_temp"`
		WaterTempError string       `json:"water_temp_error"`
		RoomTemp       float64      `json:"room_temp"`
		RoomTempError  string       `json:"room_temp_error"`
		LeakDetected   bool         `json:"leak_detected"`
		LeakError      string       `json:"leak_error"`
		PumpOn         bool         `json:"pump_on"`
		PumpError      string       `json:"pump_error"`
		PlungeStatus   PlungeStatus `json:"plunge"`
		PlungeError    string       `json:"plunge_status_error"`
		OzoneStatus    OzoneStatus  `json:"ozone"`
		OzoneError     string       `json:"ozone_status_error"`
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
		GetLatestOzone(ctx context.Context) (database.Ozone, error)
	}

	Handler struct {
		store          StatusStore
		sensors        sensor.Sensors
		state          PlungeState
		originPatterns []string
	}
)
