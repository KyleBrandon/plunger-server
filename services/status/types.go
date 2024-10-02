package status

import (
	"time"

	"github.com/KyleBrandon/plunger-server/internal/jobs"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/services/plunges/v2"
	"github.com/google/uuid"
)

type (
	OzoneStatus struct {
		StartTime       time.Time `json:"start_time"`
		EndTime         time.Time `json:"end_time"`
		Status          string    `json:"status"`
		Result          string    `json:"result"`
		ID              uuid.UUID `json:"id"`
		SecondsLeft     float64   `json:"seconds_left"`
		CancelRequested bool      `json:"cancel_requested"`
	}

	PlungeStatus struct {
		ID             uuid.UUID `json:"id,omitempty"`
		StartTime      time.Time `json:"start_time,omitempty"`
		EndTime        time.Time `json:"end_time,omitempty"`
		StartWaterTemp string    `json:"start_water_temp,omitempty"`
		StartRoomTemp  string    `json:"start_room_temp,omitempty"`
		EndWaterTemp   string    `json:"end_water_temp,omitempty"`
		EndRoomTemp    string    `json:"end_room_temp,omitempty"`

		Running          bool    `json:"running,omitempty"`
		ExpectedDuration int32   `json:"expected_duration,omitempty"`
		Remaining        float64 `json:"remaining_time,omitempty"`
		ElapsedTime      float64 `json:"elapsed_time,omitempty"`
		AvgWaterTemp     float64 `json:"average_water_temp,omitempty"`
		WaterTempError   string  `json:"water_temp_error,omitempty"`
		AvgRoomTemp      float64 `json:"average_room_temp,omitempty"`
		RoomTempError    string  `json:"room_temp_error,omitempty"`
	}

	SystemStatus struct {
		PlungeStatus PlungeStatus
		PlungeError  string `json:"plunge_status_error"`

		OzoneStatus    OzoneStatus
		OzoneError     string  `json:"ozone_status_error"`
		WaterTemp      float64 `json:"water_temp,omitempty"`
		WaterTempError string  `json:"water_temp_error,omitempty"`
		RoomTemp       float64 `json:"room_temp,omitempty"`
		RoomTempError  string  `json:"room_temp_error,omitempty"`
		LeakDetected   bool    `json:"leak_detected,omitempty"`
		LeakError      string  `json:"leak_error,omitempty"`
		PumpOn         bool    `json:"pump_on,omitempty"`
		PumpError      string  `json:"pump_error,omitempty"`
	}

	Handler struct {
		store    plunges.PlungeStore
		jobStore jobs.JobStore
		sensors  sensor.Sensors
		state    *plunges.PlungeState
	}
)
