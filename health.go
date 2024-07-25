package main

import (
	"log"
	"net/http"
)

func (config *serverConfig) handlerGetHealthz(writer http.ResponseWriter, req *http.Request) {
	log.Println("enter handlerGetHealth")
	response := struct {
		Status string `json:"status"`
	}{
		Status: "ok",
	}

	respondWithJSON(writer, http.StatusOK, response)
}
