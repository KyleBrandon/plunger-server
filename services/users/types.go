package users

import (
	"context"
	"time"

	"github.com/KyleBrandon/plunger-server/internal/database"
)

type CreateUserRequest struct {
	Email string `json:"email"`
}

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
	CreateUser(ctx context.Context, email string) (database.User, error)
}

type Handler struct {
	store UserStore
}
