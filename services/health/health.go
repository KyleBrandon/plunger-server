package health

import (
	"log/slog"
	"net/http"

	"github.com/KyleBrandon/plunger-server/utils"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (handler *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /v1/health", handler.handlerHealthGet)
}

func (handler *Handler) handlerHealthGet(writer http.ResponseWriter, req *http.Request) {
	slog.Debug("enter handlerGetHealth")
	response := struct {
		Status string `json:"status"`
	}{
		Status: "ok",
	}

	utils.RespondWithJSON(writer, http.StatusOK, response)
}
