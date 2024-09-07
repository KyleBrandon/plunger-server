package temperatures

import (
	"log/slog"
	"net/http"

	"github.com/KyleBrandon/plunger-server/utils"
)

func NewHandler(sensors Sensors) *Handler {
	return &Handler{
		sensors,
	}
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/temperatures", h.handlerTemperaturesGet)
}

func (h *Handler) handlerTemperaturesGet(w http.ResponseWriter, r *http.Request) {
	slog.Debug("handlerTemperaturesGet")

	results, err := h.sensors.ReadTemperatures()
	if err != nil {
		utils.RespondWithError(w, http.StatusInternalServerError, "failed to read temperature sensor", err)
		return
	}

	utils.RespondWithJSON(w, http.StatusOK, results)
}
