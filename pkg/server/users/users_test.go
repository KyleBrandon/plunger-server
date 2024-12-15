package users

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/pkg/utils"
)

func TestGetUser(t *testing.T) {
	userStore := mockUserStore{}

	handler := NewHandler(&userStore)

	t.Run("should fail with invalid API key format", func(t *testing.T) {
		userStore.apiKey = "12345"
		userStore.user = database.User{}

		headers := map[string][]string{
			"Authorization": {"Key 1234"},
		}
		rr := utils.TestRequestWithHeaders(t, http.MethodGet, "/v1/users", headers, nil, handler.handlerUserGet)

		if rr.Code != http.StatusForbidden {
			t.Errorf("expected status code %d, got %d", http.StatusForbidden, rr.Code)
		}
	})

	t.Run("should fail with invalid API key value", func(t *testing.T) {
		userStore.apiKey = "12345"
		userStore.user = database.User{}

		headers := map[string][]string{
			"Authorization": {"ApiKey 1234"},
		}
		rr := utils.TestRequestWithHeaders(t, http.MethodGet, "/v1/users", headers, nil, handler.handlerUserGet)

		if rr.Code != http.StatusForbidden {
			t.Errorf("expected status code %d, got %d", http.StatusForbidden, rr.Code)
		}
	})

	t.Run("should return the current user", func(t *testing.T) {
		userStore.apiKey = "12345"
		userStore.user = database.User{}

		headers := map[string][]string{
			"Authorization": {"ApiKey 12345"},
		}

		rr := utils.TestRequestWithHeaders(t, http.MethodGet, "/v1/users", headers, nil, handler.handlerUserGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}
	})
}

func TestCreateUser(t *testing.T) {
	userStore := mockUserStore{}

	handler := NewHandler(&userStore)

	t.Run("should fail with invalid request", func(t *testing.T) {
		//
		params := struct {
			Mail string `json:"name"`
		}{
			Mail: "test@mail.com",
		}

		marshalled, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}

		rr := utils.TestRequest(t, http.MethodPost, "/v1/users", bytes.NewBuffer(marshalled), handler.handlerUserCreate)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("should fail with invalid email", func(t *testing.T) {
		//
		params := CreateUserRequest{
			Email: "@mail.com",
		}

		marshalled, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}

		rr := utils.TestRequest(t, http.MethodPost, "/v1/users", bytes.NewBuffer(marshalled), handler.handlerUserCreate)
		if rr.Code != http.StatusInternalServerError {
			t.Errorf("expected status code %d, got %d", http.StatusInternalServerError, rr.Code)
		}
	})

	t.Run("should create user", func(t *testing.T) {
		//
		params := CreateUserRequest{
			Email: "test@mail.com",
		}

		marshalled, err := json.Marshal(params)
		if err != nil {
			t.Fatal(err)
		}

		rr := utils.TestRequest(t, http.MethodPost, "/v1/users", bytes.NewBuffer(marshalled), handler.handlerUserCreate)
		if rr.Code != http.StatusCreated {
			t.Errorf("expected status code %d, got %d", http.StatusCreated, rr.Code)
		}

		if userStore.user.Email != params.Email {
			t.Errorf("expected user email %s, go %s", params.Email, userStore.user.Email)
		}
	})
}

type mockUserStore struct {
	apiKey string
	user   database.User
}

func (m *mockUserStore) GetUserByApiKey(ctx context.Context, apiKey string) (database.User, error) {
	if m.apiKey != apiKey {
		return m.user, errors.New("invalid API key")
	}
	return m.user, nil
}

func (m *mockUserStore) CreateUser(ctx context.Context, email string) (database.User, error) {
	m.user.Email = email
	return m.user, nil
}
