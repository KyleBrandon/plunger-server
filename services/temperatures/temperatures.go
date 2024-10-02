package temperatures

import (
	"log/slog"
	"net/http"

	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/utils"
)

func NewHandler(sensors sensor.Sensors) *Handler {
	return &Handler{
		sensors,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/temperatures", h.handlerTemperaturesGet)
}

func (h *Handler) handlerTemperaturesGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerTemperaturesGet")

	tr := h.sensors.ReadTemperatures()

	results := make([]TemperatureReading, 0, len(tr))
	for _, t := range tr {
		results = append(results, convertFromSensorTemperatureReading(t))
	}

	slog.Info("result:", "results", results)

	utils.RespondWithJSON(w, http.StatusOK, results)
}

func convertFromSensorTemperatureReading(tr sensor.TemperatureReading) TemperatureReading {
	return TemperatureReading{
		Name:         tr.Name,
		Description:  tr.Description,
		Address:      tr.Address,
		TemperatureC: tr.TemperatureC,
		TemperatureF: tr.TemperatureF,
		Err:          tr.Err.Error(),
	}
}
