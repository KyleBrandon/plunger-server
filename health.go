package main

import (
	"log/slog"
	"net/http"
)

func (config *serverConfig) handlerHealthGet(writer http.ResponseWriter, req *http.Request) {
	slog.Debug("enter handlerGetHealth")
	response := struct {
		Status string `json:"status"`
	}{
		Status: "ok",
	}

	respondWithJSON(writer, http.StatusOK, response)
}
