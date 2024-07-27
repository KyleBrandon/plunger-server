package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/KyleBrandon/plunger-server/internal/sensor"
)

type TemperatureResponse struct {
	Address      string  `json:"id"`
	Location     string  `json:"location"`
	TemperatureC float64 `json:"temperature_c"`
	TemperatureF float64 `json:"temperature_f"`
}

func (config *serverConfig) handlerGetTemperatures(w http.ResponseWriter, r *http.Request) {
	log.Println("enter handlerGetTemperatureByLocation")

	tempSensors, err := sensor.ReadTemperatures()
	if err != nil {
		log.Printf("failed to read the temperatures: %v\n", err)
		respondWithError(w, http.StatusNotFound, "failed to read temperatures")
		return
	}

	temperatures := make([]TemperatureResponse, 0, len(tempSensors))
	for _, t := range tempSensors {
		tr := TemperatureResponse{
			Address:      t.Address,
			Location:     "",
			TemperatureC: t.TemperatureC,
			TemperatureF: t.TemperatureF,
		}

		temperatures = append(temperatures, tr)
	}

	respondWithJSON(w, http.StatusOK, temperatures)

}

func (config *serverConfig) handlerGetTemperatureByLocation(w http.ResponseWriter, r *http.Request) {
	log.Println("enter handlerGetTemperatureByLocation")
	location := r.PathValue("location")

	var temperature float64
	switch location {
	case "water":
		temperature = 0

	case "room":
		temperature = 0
	default:
		respondWithError(w, http.StatusNotFound, fmt.Sprintf("resource '%v' does not exist", location))
		return
	}

	response := TemperatureResponse{
		Address:      "",
		Location:     location,
		TemperatureC: temperature,
		TemperatureF: temperature,
	}

	respondWithJSON(w, http.StatusOK, response)
}
