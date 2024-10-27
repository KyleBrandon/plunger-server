package leaks

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/KyleBrandon/plunger-server/internal/database"
	"github.com/KyleBrandon/plunger-server/utils"
)

func TestGetCurrentLeak(t *testing.T) {
	t.Run("Fail to find a currently running leak", func(t *testing.T) {
		store := mockLeakStore{}
		h := NewHandler(&store)

		store.err = errors.New("could not find the current leak event")
		rr := utils.TestRequest(t, http.MethodGet, "/v1/leaks?filter=current", nil, h.handlerLeakGet)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected status code %d, got %d", http.StatusNotFound, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), store.err.Error()) {
			t.Errorf("expected message %s, got %s", store.err.Error(), rr.Body.String())
		}
	})

	t.Run("Succeed in finding currently running leak", func(t *testing.T) {
		store := mockLeakStore{}
		dbLeak := database.Leak{}
		store.leak = dbLeak
		h := NewHandler(&store)
		rr := utils.TestRequest(t, http.MethodGet, "/v1/leaks?filter=current", nil, h.handlerLeakGet)

		if rr.Code != http.StatusOK {
			t.Errorf("expected status code %d, got %d", http.StatusOK, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), "detected_at") {
			t.Errorf("expected message %s, got %s", "detected_at", rr.Body.String())
		}
	})
}

func TestGetAllLeaks(t *testing.T) {
	t.Run("Expect not implemented", func(t *testing.T) {
		store := mockLeakStore{}

		store.err = errors.New("read all leaks not supported")
		h := NewHandler(&store)

		rr := utils.TestRequest(t, http.MethodGet, "/v1/leaks", nil, h.handlerLeakGet)

		if rr.Code != http.StatusNotImplemented {
			t.Errorf("expected status code %d, got %d", http.StatusNotImplemented, rr.Code)
		}

		if !strings.Contains(rr.Body.String(), store.err.Error()) {
			t.Errorf("expected message %s, got %s", store.err.Error(), rr.Body.String())
		}
	})
}

type mockLeakStore struct {
	err  error
	leak database.Leak
}

func (m *mockLeakStore) GetLatestLeakDetected(ctx context.Context) (database.Leak, error) {
	return m.leak, m.err
}
