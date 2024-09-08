package users

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/mail"

	"github.com/KyleBrandon/plunger-server/internal/auth"
	"github.com/KyleBrandon/plunger-server/utils"
)

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

func validateEmail(email string) bool {
	_, err := mail.ParseAddress(email)
	return err == nil
}

func (handler *Handler) handlerUserCreate(writer http.ResponseWriter, req *http.Request) {
	slog.Debug("handleUserCreate")

	var params CreateUserRequest

	decoder := json.NewDecoder(req.Body)
	err := decoder.Decode(&params)
	if err != nil {
		utils.RespondWithError(writer, http.StatusInternalServerError, "could not parse body", err)
		return
	}

	if !validateEmail(params.Email) {
		slog.Debug(fmt.Sprintf("create user attempted with invalid email %s\n", params.Email))
		utils.RespondWithError(writer, http.StatusInternalServerError, "could not parse body", err)
		return
	}

	ctx := context.Background()

	user, err := handler.store.CreateUser(ctx, params.Email)
	if err != nil {
		utils.RespondWithError(writer, http.StatusInternalServerError, "failed to create user", err)
		return
	}

	utils.RespondWithJSON(writer, http.StatusCreated, databaseUserToUser(user))
}
