package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/google/uuid"
)

type PlungeResponse struct {
	ID             uuid.UUID `json:"id,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	UpdatedAt      time.Time `json:"updated_at,omitempty"`
	StartTime      time.Time `json:"start_time,omitempty"`
	EndTime        time.Time `json:"end_time,omitempty"`
	Running        bool      `json:"running"`
	ElapsedTime    float64   `json:"elapsed_time"`
	StartRoomTemp  string    `json:"start_room_temp,omitempty"`
	EndRoomTemp    string    `json:"end_room_temp,omitempty"`
	StartWaterTemp string    `json:"start_water_temp,omitempty"`
	EndWaterTemp   string    `json:"end_water_temp,omitempty"`
}

func databasePlungeToPlunge(dbPlunge database.Plunge) PlungeResponse {

	resp := PlungeResponse{
		ID:        dbPlunge.ID,
		CreatedAt: dbPlunge.CreatedAt,
		UpdatedAt: dbPlunge.UpdatedAt,
	}

	if dbPlunge.StartTime.Valid {
		resp.StartTime = dbPlunge.StartTime.Time
	}
	if dbPlunge.EndTime.Valid {
		resp.EndTime = dbPlunge.EndTime.Time
		resp.ElapsedTime = resp.EndTime.Sub(resp.StartTime).Seconds()
		resp.Running = false
	} else {
		resp.ElapsedTime = time.Now().UTC().Sub(resp.StartTime).Seconds()
		resp.Running = true
	}
	if dbPlunge.StartWaterTemp.Valid {
		resp.StartWaterTemp = dbPlunge.StartWaterTemp.String
	}
	if dbPlunge.EndWaterTemp.Valid {
		resp.EndWaterTemp = dbPlunge.EndWaterTemp.String
	}
	if dbPlunge.StartRoomTemp.Valid {
		resp.StartRoomTemp = dbPlunge.StartRoomTemp.String
	}
	if dbPlunge.EndRoomTemp.Valid {
		resp.EndRoomTemp = dbPlunge.EndRoomTemp.String
	}

	return resp
}

func (config *serverConfig) handlePlungesGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesGet")
	dbPlunges := make([]database.Plunge, 0)

	plungeID := r.PathValue("PLUNGE_ID")
	if plungeID != "" {
		pid, err := uuid.Parse(plungeID)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "could not find plunge", err)
			return
		}

		p, err := config.DB.GetPlungeByID(r.Context(), pid)
		if err != nil {
			respondWithError(w, http.StatusNotFound, "could not find plunge", err)
			return
		}
		dbPlunges = append(dbPlunges, p)
	} else {
		filter := r.URL.Query().Get("filter")
		if filter == "current" {
			p, err := config.DB.GetLatestPlunge(r.Context())
			if err != nil {
				respondWithError(w, http.StatusNotFound, "could not find a current plunge", err)
				return
			}

			dbPlunges = append(dbPlunges, p)
		} else {
			p, err := config.DB.GetPlunges(r.Context())
			if err != nil {
				respondWithError(w, http.StatusNotFound, "could not find any plunges", err)
				return
			}

			dbPlunges = append(dbPlunges, p...)
		}
	}

	plunges := make([]PlungeResponse, 0, len(dbPlunges))
	for _, p := range dbPlunges {
		plunges = append(plunges, databasePlungeToPlunge(p))
	}

	respondWithJSON(w, http.StatusOK, plunges)
}

func (config *serverConfig) handlePlungesStart(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesStart")

	temperatures, err := config.Sensors.ReadTemperatures()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	waterTemp := sql.NullString{Valid: false}
	roomTemp := sql.NullString{Valid: false}

	for _, temp := range temperatures {
		if temp.Name == "Room" {
			roomTemp.Valid = true
			roomTemp.String = fmt.Sprintf("%f", temp.TemperatureF)
		} else if temp.Name == "Water" {
			waterTemp.Valid = true
			waterTemp.String = fmt.Sprintf("%f", temp.TemperatureF)
		}
	}

	params := database.StartPlungeParams{
		StartTime:      sql.NullTime{Valid: true, Time: time.Now().UTC()},
		StartWaterTemp: waterTemp,
		StartRoomTemp:  roomTemp,
	}

	plunge, err := config.DB.StartPlunge(r.Context(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	respondWithJSON(w, http.StatusCreated, databasePlungeToPlunge(plunge))
}

func (config *serverConfig) handlePlungesStop(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesStop")

	plungeID := r.PathValue("PLUNGE_ID")
	if plungeID == "" {
		respondWithError(w, http.StatusNotFound, "could not find plunge", errors.New("plunge id path value not set"))
		return
	}
	pid, err := uuid.Parse(plungeID)
	if err != nil {
		respondWithError(w, http.StatusNotFound, "could not find plunge", err)
		return
	}

	temperatures, err := config.Sensors.ReadTemperatures()
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	waterTemp := sql.NullString{Valid: false}
	roomTemp := sql.NullString{Valid: false}

	for _, temp := range temperatures {
		if temp.Name == "Room" {
			roomTemp.Valid = true
			roomTemp.String = fmt.Sprintf("%f", temp.TemperatureF)
		} else if temp.Name == "Water" {
			waterTemp.Valid = true
			waterTemp.String = fmt.Sprintf("%f", temp.TemperatureF)
		}
	}

	params := database.StopPlungeParams{
		ID:           pid,
		EndTime:      sql.NullTime{Valid: true, Time: time.Now().UTC()},
		EndWaterTemp: waterTemp,
		EndRoomTemp:  roomTemp,
	}

	_, err = config.DB.StopPlunge(r.Context(), params)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	respondWithNoContent(w, http.StatusNoContent)
}
