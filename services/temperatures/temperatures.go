package temperatures

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/utils"
)

func NewHandler(store TemperatureStore, sensors sensor.Sensors) *Handler {
	return &Handler{
		store,
		sensors,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	go h.monitorTemperatures(context.Background())

	mux.HandleFunc("GET /v1/temperatures", h.handlerTemperaturesGet)
}

func (h *Handler) handlerTemperaturesGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerTemperaturesGet")

	tr := h.sensors.ReadTemperatures()

	results := make([]TemperatureReading, 0, len(tr))
	for _, t := range tr {
		results = append(results, convertFromSensorTemperatureReading(t))
	}

	utils.RespondWithJSON(w, http.StatusOK, results)
}

func convertFromSensorTemperatureReading(tr sensor.TemperatureReading) TemperatureReading {
	errorMessage := ""
	if tr.Err != nil {
		errorMessage = tr.Err.Error()
	}
	return TemperatureReading{
		Name:         tr.Name,
		Description:  tr.Description,
		Address:      tr.Address,
		TemperatureC: tr.TemperatureC,
		TemperatureF: tr.TemperatureF,
		Err:          errorMessage,
	}
}

func (h *Handler) monitorTemperatures(ctx context.Context) {
	for {
		waterTemp := sql.NullString{
			Valid: false,
		}
		roomTemp := sql.NullString{
			Valid: false,
		}

		rt, wt := h.sensors.ReadRoomAndWaterTemperature()
		if rt.Err == nil {
			roomTemp.Valid = true
			roomTemp.String = fmt.Sprintf("%f", rt.TemperatureF)
		} else {
			slog.Error("failed to read the room temperature", "error", rt.Err)
		}

		if wt.Err == nil {
			waterTemp.Valid = true
			waterTemp.String = fmt.Sprintf("%f", wt.TemperatureF)
		} else {
			slog.Error("failed to read the water temperature", "error", wt.Err)
		}

		arg := database.SaveTemperatureParams{
			WaterTemp: waterTemp,
			RoomTemp:  roomTemp,
		}
		_, err := h.store.SaveTemperature(ctx, arg)
		if err != nil {
			slog.Error("failed to save the room and water temperatures", "error", err)
		}

		time.Sleep(30 * time.Second)
	}
}
