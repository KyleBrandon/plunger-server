package main

import (
	"encoding/json"
	"io"
	"net/http"
)

func respondWithJSON(writer http.ResponseWriter, code int, payload interface{}) {
	resultData, err := json.Marshal(payload)
	if err != nil {
		respondWithError(writer, http.StatusBadRequest, "Error marshalling result")
		return

	}

	writer.Header().Set("Content-Type", "application/json")
	writer.WriteHeader(code)
	writer.Write(resultData)
}

func respondWithError(writer http.ResponseWriter, code int, error string) {
	response := struct {
		Error string `json:"error"`
	}{
		Error: error,
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
