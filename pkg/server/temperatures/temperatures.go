package temperatures

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"

	"github.com/KyleBrandon/plunger-server/internal/sensor"
	"github.com/KyleBrandon/plunger-server/pkg/server/monitor"
	"github.com/KyleBrandon/plunger-server/pkg/utils"
)

func NewHandler(mctx *monitor.MonitorContext, sensors sensor.Sensors) *Handler {
	return &Handler{
		mctx,
		sensors,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/temperatures", h.handlerTemperaturesGet)
	mux.HandleFunc("POST /v1/temperatures/notify", h.handerTemperatureNotify)
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

func (h *Handler) handerTemperatureNotify(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid body for temperature notification", err)
		return
	}

	defer r.Body.Close()

	var tnr TemperatureNotifyRequest
	if err := json.Unmarshal(body, &tnr); err != nil {
		utils.RespondWithError(w, http.StatusBadRequest, "Invalid body for temperature notification", err)
		return
	}

	h.mctx.TempMonitorCh <- monitor.TemperatureTask{TargetTemperature: tnr.TargetTemperature}

	utils.RespondWithNoContent(w, http.StatusCreated)
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
