package plunges

import (
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/utils"
	"github.com/google/uuid"
)

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
	resp.StartWaterTemp = dbPlunge.StartWaterTemp
	resp.EndWaterTemp = dbPlunge.EndWaterTemp
	resp.StartRoomTemp = dbPlunge.StartRoomTemp
	resp.EndRoomTemp = dbPlunge.EndRoomTemp

	return resp
}

func NewHandler(store PlungeStore, sensors Sensors) *Handler {
	return &Handler{
		store,
		sensors,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/plunges", h.handlePlungesGet)
	mux.HandleFunc("POST /v1/plunges", h.handlePlungesStart)
	mux.HandleFunc("PUT /v1/plunges/{PLUNGE_ID}", h.handlePlungesStop)
}

func (h *Handler) handlePlungesGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesGet")
	dbPlunges := make([]database.Plunge, 0)

	// TODO: support appropriate get all and get w/id
	plungeID := strings.TrimPrefix(r.URL.Path, "/v1/plunges/")
	if plungeID != "" && plungeID != r.URL.Path {
		pid, err := uuid.Parse(plungeID)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "could not find plunge", err)
			return
		}

		p, err := h.store.GetPlungeByID(r.Context(), pid)
		if err != nil {
			utils.RespondWithError(w, http.StatusNotFound, "could not find plunge", err)
			return
		}
		dbPlunges = append(dbPlunges, p)
	} else {
		filter := r.URL.Query().Get("filter")
		if filter == "current" {
			p, err := h.store.GetLatestPlunge(r.Context())
			if err != nil {
				utils.RespondWithError(w, http.StatusNotFound, "could not find a current plunge", err)
				return
			}

			dbPlunges = append(dbPlunges, p)
		} else {
			p, err := h.store.GetPlunges(r.Context())
			if err != nil {
				utils.RespondWithError(w, http.StatusNotFound, "could not find any plunges", err)
				return
			}

			dbPlunges = append(dbPlunges, p...)
		}
	}

	plunges := make([]PlungeResponse, 0, len(dbPlunges))
	for _, p := range dbPlunges {
		plunges = append(plunges, databasePlungeToPlunge(p))
	}

	utils.RespondWithJSON(w, http.StatusOK, plunges)
}

func (h *Handler) handlePlungesStart(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesStart")

	temperatures, err := h.sensors.ReadTemperatures()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	waterTemp := ""
	roomTemp := ""

	for _, temp := range temperatures {
		if temp.Name == "Room" {
			roomTemp = fmt.Sprintf("%f", temp.TemperatureF)
		} else if temp.Name == "Water" {
			waterTemp = fmt.Sprintf("%f", temp.TemperatureF)
		}
	}

	params := database.StartPlungeParams{
		StartTime:      sql.NullTime{Valid: true, Time: time.Now().UTC()},
		StartWaterTemp: waterTemp,
		StartRoomTemp:  roomTemp,
	}

	plunge, err := h.store.StartPlunge(r.Context(), params)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to start the plunge timer", err)
		return
	}

	utils.RespondWithJSON(w, http.StatusCreated, databasePlungeToPlunge(plunge))
}

func (h *Handler) handlePlungesStop(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlePlungesStop")

	plungeID := r.PathValue("PLUNGE_ID")
	if plungeID == "" {
		utils.RespondWithError(w, http.StatusNotFound, "could not find plunge", errors.New("plunge id path value not set"))
		return
	}
	pid, err := uuid.Parse(plungeID)
	if err != nil {
		utils.RespondWithError(w, http.StatusNotFound, "could not find plunge", err)
		return
	}

	temperatures, err := h.sensors.ReadTemperatures()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to read the temperature at stop plunge", err)
		return
	}

	waterTemp := ""
	roomTemp := ""

	for _, temp := range temperatures {
		if temp.Name == "Room" {
			roomTemp = fmt.Sprintf("%f", temp.TemperatureF)
		} else if temp.Name == "Water" {
			waterTemp = fmt.Sprintf("%f", temp.TemperatureF)
		}
	}

	params := database.StopPlungeParams{
		ID:           pid,
		EndTime:      sql.NullTime{Valid: true, Time: time.Now().UTC()},
		EndWaterTemp: waterTemp,
		EndRoomTemp:  roomTemp,
		AvgWaterTemp: "0.0",
		AvgRoomTemp:  "0.0",
	}

	_, err = h.store.StopPlunge(r.Context(), params)
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to update the plunge to stop", err)
		return
	}

	utils.RespondWithNoContent(w, http.StatusNoContent)
}
