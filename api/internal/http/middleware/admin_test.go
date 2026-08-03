package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminAuth(t *testing.T) {
	nextCalled := false
	next := AdminAuth("admin-token-must-be-at-least-32-bytes")(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		nextCalled = true
	}))
	missing := httptest.NewRecorder()
	next.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/v1/admin/overview", nil))
	if missing.Code != http.StatusUnauthorized || nextCalled {
		t.Fatalf("missing token: %d next=%v", missing.Code, nextCalled)
	}
	authorized := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/v1/admin/overview", nil)
	request.Header.Set("Authorization", "Bearer admin-token-must-be-at-least-32-bytes")
	next.ServeHTTP(authorized, request)
	if authorized.Code != http.StatusOK || !nextCalled {
		t.Fatalf("authorized token: %d next=%v", authorized.Code, nextCalled)
	}
}
