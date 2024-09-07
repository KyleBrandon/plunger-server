package users

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/auth"
	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/utils"
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

type UserStore interface {
	GetUserByApiKey(ctx context.Context, apiKey string) (database.User, error)
	CreateUser(ctx context.Context, args database.CreateUserParams) (database.User, error)
}

type Handler struct {
	store UserStore
}

func NewHandler(store UserStore) *Handler {
	return &Handler{
		store: store,
	}
}

func (handler *Handler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /v1/users", handler.handlerUserCreate)
	mux.HandleFunc("GET /v1/users", handler.handlerUserGet)
}

func (handler *Handler) handlerUserGet(writer http.ResponseWriter, req *http.Request) {
	slog.Debug("handleUserGet")

	apiKey, err := auth.ParseApiKey(req)
	if err != nil {
		utils.RespondWithError(writer, http.StatusForbidden, "not authorized", err)
		return
	}

	user, err := handler.store.GetUserByApiKey(req.Context(), apiKey)
	if err != nil {
		utils.RespondWithError(writer, http.StatusForbidden, "not authorized", err)
		return
	}

	utils.RespondWithJSON(writer, http.StatusOK, databaseUserToUser(user))
}

func (handler *Handler) handlerUserCreate(writer http.ResponseWriter, req *http.Request) {
	slog.Debug("handleUserCreate")

	params := struct {
		Email string `json:"email"`
	}{}

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		utils.RespondWithError(writer, http.StatusInternalServerError, "could not parse body", err)
		return
	}

	ctx := context.Background()

	createUserParams := database.CreateUserParams{
		ID:        uuid.New(),
		Email:     params.Email,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	user, err := handler.store.CreateUser(ctx, createUserParams)
	if err != nil {
		utils.RespondWithError(writer, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	utils.RespondWithJSON(writer, http.StatusCreated, databaseUserToUser(user))
}
