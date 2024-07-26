package auth

import (
	"errors"
	"fmt"
	"net/http"
)

var ErrNoAuthHeader = errors.New("authorization header not found")

func ParseApiKey(r *http.Request) (string, error) {

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return "", ErrNoAuthHeader
	}

	var apiKey string
	n, err := fmt.Sscanf(authHeader, "ApiKey %s", &apiKey)
	if n != 1 || err != nil {
		return "", ErrNoAuthHeader
	}

	return apiKey, nil
}
