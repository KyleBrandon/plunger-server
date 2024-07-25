package main

import "net/http"

func (config *serverConfig) handlerGetHealth(writer http.ResponseWriter, req *http.Request) {
	response := struct {
		Status string `json:"status"`
	}{
		Status: "ok",
	}

	respondWithJSON(writer, http.StatusOK, response)
}
