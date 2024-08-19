package main

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
)

func respondWithJSON(writer http.ResponseWriter, code int, payload interface{}) {
	resultData, err := json.Marshal(payload)
	if err != nil {
		respondWithError(writer, http.StatusBadRequest, "Error marshalling result", err)
		return
	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	writer.Write(resultData)
}

func respondWithError(writer http.ResponseWriter, code int, message string, err error) {
	slog.Error(message, "http_status", code, "error", err)

	response := struct {
		Error string `json:"error"`
	}{
		Error: message,
	}

	respondWithJSON(writer, code, response)
}

func respondWithString(writer http.ResponseWriter, contentType string, code int, msg string) {
	writer.Header().Set("Content-Type", contentType)
	writer.WriteHeader(code)
	io.WriteString(writer, msg)
}

func respondWithNoContent(writer http.ResponseWriter, code int) {
	writer.WriteHeader(code)
}
