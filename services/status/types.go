package status

import (
	"sync"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/services/plunges/v2"
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
		WaterTempError   string  `json:"water_temp_error"`
		AvgRoomTemp      float64 `json:"average_room_temp"`
		RoomTempError    string  `json:"room_temp_error"`
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

	Handler struct {
		store          plunges.PlungeStore
		jobStore       jobs.JobStore
		sensors        sensor.Sensors
		state          PlungeState
		originPatterns []string
	}
)
