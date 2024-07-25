package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/google/uuid"
)

func (config *serverConfig) handlerCreateUser(writer http.ResponseWriter, req *http.Request) {
	params := struct {
		Email string `json:"email"`
	}{}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		log.Printf("Could not parse body: %v\n", err)
		respondWithError(writer, http.StatusInternalServerError, "could not parse body")
		return
	}

	ctx := context.Background()

	createUserParams := database.CreateUserParams{
		ID:        uuid.New(),
		Email:     params.Email,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	user, err := config.DB.CreateUser(ctx, createUserParams)
	if err != nil {
		log.Printf("Failed to create user: %v\n", err)
		respondWithError(writer, http.StatusInternalServerError, "failed to create user")
		return
	}

	userResponse := struct {
		ID        string    `json:"id"`
		Email     string    `json:"email"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
	}{
		ID:        user.ID.String(),
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}

	respondWithJSON(writer, http.StatusCreated, userResponse)
}
