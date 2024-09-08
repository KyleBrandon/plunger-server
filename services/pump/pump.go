package pump

import (
	"log/slog"
	"net/http"

	"github.com/KyleBrandon/plunger-server/utils"
)

func NewHandler(pump PumpSensor) *Handler {
	return &Handler{
		pump,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/pump", h.handlerPumpGet)
	mux.HandleFunc("POST /v1/pump/start", h.handlerPumpStart)
	mux.HandleFunc("POST /v1/pump/stop", h.handlerPumpStop)
}

func (h *Handler) handlerPumpGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerPumpGet")

	pumpOn, err := h.pump.IsPumpOn()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "could not start the ozone timer", err)
		return
	}

	response := struct {
		PumpOn bool `json:"pump_on"`
	}{
		PumpOn: pumpOn,
	}

	utils.RespondWithJSON(w, http.StatusOK, response)
}

func (h *Handler) handlerPumpStart(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerPumpStart")
	err := h.pump.TurnPumpOn()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to turn on the pump", err)
		return
	}

	utils.RespondWithNoContent(w, http.StatusNoContent)
}

func (h *Handler) handlerPumpStop(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerPumpStop")

	err := h.pump.TurnPumpOff()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to turn off the pump", err)
		return
	}

	utils.RespondWithNoContent(w, http.StatusNoContent)
}
