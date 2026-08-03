package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"deinscomplete/api/internal/account"
	"deinscomplete/api/internal/auth"
	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/completion/providers"
	"deinscomplete/api/internal/usage"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type providerErrorProvider struct{}

func (providerErrorProvider) Complete(context.Context, completion.Request) (completion.Result, error) {
	return completion.Result{}, completion.NewProviderError(completion.ProviderTimeout, context.DeadlineExceeded)
}

func testRouter() http.Handler {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	return newRouter(logger, completion.NewService(providers.MockProvider{}), auth.New("", 1, 0), false, false, nil, nil, nil, nil, nil, nil, nil, false, "")
}

func TestHealthAndReady(t *testing.T) {
	for _, path := range []string{"/health", "/ready"} {
		response := httptest.NewRecorder()
		testRouter().ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("%s: got %d", path, response.Code)
		}
	}
}

func TestCompletionAndRequestID(t *testing.T) {
	body := `{"context":{"prefix":"const user =","suffix":"","language":"typescript","filePath":"/test.ts","cursorOffset":12}}`
	request := httptest.NewRequest(http.MethodPost, "/v1/completions", strings.NewReader(body))
	request.Header.Set("X-Request-ID", "test-request")
	response := httptest.NewRecorder()
	testRouter().ServeHTTP(response, request)
	if response.Code != http.StatusOK || response.Header().Get("X-Request-ID") != "test-request" {
		t.Fatalf("unexpected response: %d", response.Code)
	}
	var payload struct {
		Completion struct {
			Text string `json:"text"`
		} `json:"completion"`
		Metadata struct {
			RequestID string `json:"requestId"`
		} `json:"metadata"`
	}
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil || payload.Completion.Text != "await getUser();" || payload.Metadata.RequestID != "test-request" {
		t.Fatalf("unexpected payload: %#v, %v", payload, err)
	}
}

func TestUnknownCompletionReturnsEmptyText(t *testing.T) {
	body := `{"context":{"prefix":"let value =","suffix":"","language":"typescript","filePath":"/test.ts","cursorOffset":11}}`
	response := httptest.NewRecorder()
	testRouter().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/completions", strings.NewReader(body)))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), `"text":""`) {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
}

func TestCompletionValidationAndUnknownRoute(t *testing.T) {
	cases := []struct {
		name, path, body string
		status           int
	}{
		{"invalid json", "/v1/completions", "{", http.StatusBadRequest},
		{"missing context", "/v1/completions", `{}`, http.StatusBadRequest},
		{"empty language", "/v1/completions", `{"context":{"prefix":"","suffix":"","language":"","filePath":"","cursorOffset":0}}`, http.StatusBadRequest},
		{"negative cursor", "/v1/completions", `{"context":{"prefix":"","suffix":"","language":"ts","filePath":"","cursorOffset":-1}}`, http.StatusBadRequest},
		{"absolute repository path", "/v1/completions", `{"context":{"prefix":"","suffix":"","language":"ts","filePath":"","cursorOffset":0},"repositoryContext":{"files":[{"path":"/private/key.ts","language":"ts","content":"export const x=1","reason":"import"}]}}`, http.StatusBadRequest},
		{"oversized repository files", "/v1/completions", `{"context":{"prefix":"","suffix":"","language":"ts","filePath":"","cursorOffset":0},"repositoryContext":{"files":[{"path":"a.ts","language":"ts","content":"x","reason":"import"},{"path":"b.ts","language":"ts","content":"x","reason":"import"},{"path":"c.ts","language":"ts","content":"x","reason":"import"},{"path":"d.ts","language":"ts","content":"x","reason":"import"},{"path":"e.ts","language":"ts","content":"x","reason":"import"},{"path":"f.ts","language":"ts","content":"x","reason":"import"},{"path":"g.ts","language":"ts","content":"x","reason":"import"},{"path":"h.ts","language":"ts","content":"x","reason":"import"},{"path":"i.ts","language":"ts","content":"x","reason":"import"}]}}`, http.StatusBadRequest},
		{"unknown route", "/unknown", "", http.StatusNotFound},
	}
	for _, testCase := range cases {
		response := httptest.NewRecorder()
		testRouter().ServeHTTP(response, httptest.NewRequest(http.MethodPost, testCase.path, strings.NewReader(testCase.body)))
		if response.Code != testCase.status || !strings.Contains(response.Body.String(), `"requestId"`) {
			t.Fatalf("%s: got %d %s", testCase.name, response.Code, response.Body.String())
		}
	}
}

func TestMethodNotAllowedReturnsJSON(t *testing.T) {
	response := httptest.NewRecorder()
	testRouter().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/v1/completions", nil))
	if response.Code != http.StatusMethodNotAllowed || !strings.Contains(response.Body.String(), `"METHOD_NOT_ALLOWED"`) {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
}

func TestProviderErrorsUseSafeAPIResponses(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(logger, completion.NewService(providerErrorProvider{}), auth.New("", 1, 0), false, false, nil, nil, nil, nil, nil, nil, nil, false, "")
	body := `{"context":{"prefix":"const user =","suffix":"","language":"typescript","filePath":"test.ts","cursorOffset":12}}`
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/completions", strings.NewReader(body)))
	if response.Code != http.StatusGatewayTimeout || !strings.Contains(response.Body.String(), `"PROVIDER_TIMEOUT"`) {
		t.Fatalf("unexpected response: %d %s", response.Code, response.Body.String())
	}
}

func TestOversizedCompletionBody(t *testing.T) {
	response := httptest.NewRecorder()
	testRouter().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/completions", bytes.NewReader(make([]byte, 256*1024+1))))
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %d", response.Code)
	}
}

func TestAuthenticatedCompletionAndRegistration(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := auth.New("01234567890123456789012345678901", 1, 0)
	router := newRouter(logger, completion.NewService(providers.MockProvider{}), service, true, false, nil, nil, nil, nil, nil, nil, nil, false, "")
	registration := httptest.NewRecorder()
	router.ServeHTTP(registration, httptest.NewRequest(http.MethodPost, "/v1/installations/register", strings.NewReader(`{"installationId":"installation-1"}`)))
	if registration.Code != http.StatusOK {
		t.Fatalf("registration: %d", registration.Code)
	}
	var payload struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(registration.Body).Decode(&payload); err != nil || payload.Token == "" {
		t.Fatalf("invalid registration response: %v", err)
	}
	body := `{"context":{"prefix":"const user =","suffix":"","language":"typescript","filePath":"test.ts","cursorOffset":12}}`
	missing := httptest.NewRecorder()
	router.ServeHTTP(missing, httptest.NewRequest(http.MethodPost, "/v1/completions", strings.NewReader(body)))
	if missing.Code != http.StatusUnauthorized {
		t.Fatalf("missing auth: %d", missing.Code)
	}
	authorized := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/completions", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+payload.Token)
	router.ServeHTTP(authorized, req)
	if authorized.Code != http.StatusOK {
		t.Fatalf("authorized completion: %d", authorized.Code)
	}
	headerAuthorized := httptest.NewRecorder()
	headerRequest := httptest.NewRequest(http.MethodPost, "/v1/completions", strings.NewReader(body))
	headerRequest.Header.Set("Authorization", "Bearer invalid-token")
	headerRequest.Header.Set("X-DeinsComplete-Installation-Token", payload.Token)
	router.ServeHTTP(headerAuthorized, headerRequest)
	if headerAuthorized.Code != http.StatusOK {
		t.Fatalf("installation-header authorized completion: %d", headerAuthorized.Code)
	}
}

func TestLinkedInstallationReachesCompletionAfterEntitlementResolution(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo := account.NewRepository(pool)
	user, err := repo.CreateUser(context.Background(), account.CreateUserParams{Email: uuid.NewString() + "@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	installation, err := repo.EnsureInstallation(context.Background(), uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.LinkInstallation(context.Background(), installation.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM installations WHERE id=$1`, installation.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM user_entitlements WHERE user_id=$1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, user.ID)
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	authService := auth.New("01234567890123456789012345678901", 1, 0)
	router := newRouter(logger, completion.NewService(providers.MockProvider{}), authService, true, true, nil, nil, usage.NewMonthly(), nil, repo, nil, nil, true, "")
	token, err := authService.Issue(installation.InstallationKey)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{"/v1/completions", "/v1/completions/stream"} {
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{"context":{"prefix":"const user =","suffix":"","language":"typescript","filePath":"test.ts","cursorOffset":12}}`))
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("%s: %d %s", path, response.Code, response.Body.String())
		}
	}
}

func TestAdminPanelRoutes(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not configured")
	}
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	repo := account.NewRepository(pool)
	user, err := repo.CreateUser(context.Background(), account.CreateUserParams{Email: uuid.NewString() + "@example.test"})
	if err != nil {
		t.Fatal(err)
	}
	installation, err := repo.EnsureInstallation(context.Background(), uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if _, err = repo.LinkInstallation(context.Background(), installation.ID, user.ID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM installations WHERE id=$1`, installation.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM invites WHERE email_normalized LIKE '%@example.test'`)
		_, _ = pool.Exec(context.Background(), `DELETE FROM user_entitlements WHERE user_id=$1`, user.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, user.ID)
	})

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(logger, completion.NewService(providers.MockProvider{}), auth.New("", 1, 0), false, false, nil, nil, usage.NewMonthly(), nil, repo, nil, nil, false, "admin-token-must-be-at-least-32-bytes")
	unauthorized := httptest.NewRecorder()
	router.ServeHTTP(unauthorized, httptest.NewRequest(http.MethodGet, "/v1/admin/overview", nil))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("admin without token: %d", unauthorized.Code)
	}
	do := func(method, path, body string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(method, path, strings.NewReader(body))
		request.Header.Set("Authorization", "Bearer admin-token-must-be-at-least-32-bytes")
		if body != "" {
			request.Header.Set("Content-Type", "application/json")
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response
	}
	if response := do(http.MethodGet, "/v1/admin/overview", ""); response.Code != http.StatusOK {
		t.Fatalf("overview: %d %s", response.Code, response.Body.String())
	}
	if response := do(http.MethodPost, "/v1/admin/invites", `{"email":"invite@example.test","days":3}`); response.Code != http.StatusCreated || !strings.Contains(response.Body.String(), `"code"`) {
		t.Fatalf("create invite: %d %s", response.Code, response.Body.String())
	}
	if response := do(http.MethodPost, "/v1/admin/users/"+user.ID+"/plan", `{"plan":"pro"}`); response.Code != http.StatusOK {
		t.Fatalf("set plan: %d %s", response.Code, response.Body.String())
	}
	if response := do(http.MethodPost, "/v1/admin/installations/"+installation.ID+"/revoke", ""); response.Code != http.StatusNoContent {
		t.Fatalf("revoke installation: %d %s", response.Code, response.Body.String())
	}
}

func TestStreamingCompletionEndpoint(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	router := newRouter(logger, completion.NewService(providers.MockProvider{}), auth.New("", 1, 0), false, true, nil, nil, nil, nil, nil, nil, nil, false, "")
	body := `{"context":{"prefix":"const user =","suffix":"","language":"typescript","filePath":"test.ts","cursorOffset":12}}`
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/completions/stream", strings.NewReader(body)))
	if response.Code != http.StatusOK || !strings.Contains(response.Header().Get("Content-Type"), "text/event-stream") || !strings.Contains(response.Body.String(), "event: done") {
		t.Fatalf("unexpected stream: %d %s", response.Code, response.Body.String())
	}
}
