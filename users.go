package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/auth"
	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/google/uuid"
)

type UserResponse struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	ApiKey    string    `json:"api_key"`
}

func newUserResponse(user database.User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		ApiKey:    user.ApiKey,
	}
}

func (config *serverConfig) handlerGetUser(writer http.ResponseWriter, req *http.Request) {
	apiKey, err := auth.ParseApiKey(req)
	if err != nil {
		log.Printf("Could not parse the API key: %v\n", err)
		respondWithError(writer, http.StatusForbidden, "not authorized")
		return
	}

	user, err := config.DB.GetUserByApiKey(req.Context(), apiKey)
	if err != nil {
		log.Printf("Could not find user with API key: %v\n", err)
		respondWithError(writer, http.StatusForbidden, "not authorized")
		return
	}

	respondWithJSON(writer, http.StatusOK, newUserResponse(user))
}

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

	respondWithJSON(writer, http.StatusCreated, newUserResponse(user))
}
