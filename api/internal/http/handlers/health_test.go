package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReadyReflectsDependencyStatus(t *testing.T) {
	healthy := httptest.NewRecorder()
	Ready(nil).ServeHTTP(healthy, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if healthy.Code != http.StatusOK {
		t.Fatalf("ready status: %d", healthy.Code)
	}

	unavailable := httptest.NewRecorder()
	Ready(func(_ context.Context) error { return errors.New("unavailable") }).
		ServeHTTP(unavailable, httptest.NewRequest(http.MethodGet, "/ready", nil))
	if unavailable.Code != http.StatusServiceUnavailable {
		t.Fatalf("unavailable status: %d", unavailable.Code)
	}
}
