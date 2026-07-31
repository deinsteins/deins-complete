package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"deinscomplete/api/internal/ratelimit"
	"deinscomplete/api/internal/usage"
)

type failingLimiter struct{}

func (failingLimiter) Allow(context.Context, string) ratelimit.Result {
	return ratelimit.Result{Err: errors.New("storage unavailable")}
}

type failingTracker struct{}

func (failingTracker) CheckAndConsume(context.Context, string, int) usage.Result {
	return usage.Result{Err: errors.New("storage unavailable")}
}

func TestAdmissionStorageFailureReturnsServiceUnavailable(t *testing.T) {
	tests := []struct {
		name       string
		middleware func(http.Handler) http.Handler
	}{
		{"rate limit", RateLimit(failingLimiter{})},
		{"quota", Quota(failingTracker{})},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			nextCalled := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { nextCalled = true })
			request := httptest.NewRequest(http.MethodPost, "/v1/completions", nil)
			request = request.WithContext(context.WithValue(request.Context(), identityKey{}, Identity{InstallationID: "installation"}))
			responseRecorder := httptest.NewRecorder()

			testCase.middleware(next).ServeHTTP(responseRecorder, request)

			if responseRecorder.Code != http.StatusServiceUnavailable ||
				!strings.Contains(responseRecorder.Body.String(), `"SERVICE_UNAVAILABLE"`) ||
				nextCalled {
				t.Fatalf("unexpected response: %d %s next=%v", responseRecorder.Code, responseRecorder.Body.String(), nextCalled)
			}
		})
	}
}
