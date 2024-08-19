package main

import (
	"context"
	"encoding/json"
	"log/slog"
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

func databaseUserToUser(user database.User) UserResponse {
	return UserResponse{
		ID:        user.ID.String(),
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		ApiKey:    user.ApiKey,
	}
}

func (config *serverConfig) handlerUserGet(writer http.ResponseWriter, req *http.Request) {
	slog.Debug("handleUserGet")

	apiKey, err := auth.ParseApiKey(req)
	if err != nil {
		respondWithError(writer, http.StatusForbidden, "not authorized", err)
		return
	}

	user, err := config.DB.GetUserByApiKey(req.Context(), apiKey)
	if err != nil {
		respondWithError(writer, http.StatusForbidden, "not authorized", err)
		return
	}

	respondWithJSON(writer, http.StatusOK, databaseUserToUser(user))
}

func (config *serverConfig) handlerUserCreate(writer http.ResponseWriter, req *http.Request) {
	slog.Debug("handleUserCreate")

	params := struct {
		Email string `json:"email"`
	}{}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, "could not parse body", err)
		return
	}

	ctx := context.Background()

	createUserParams := database.CreateUserParams{
		ID:        uuid.New(),
		Email:     params.Email,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	user, err := config.DB.CreateUser(ctx, createUserParams)
	if err != nil {
		respondWithError(writer, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	respondWithJSON(writer, http.StatusCreated, databaseUserToUser(user))
}
