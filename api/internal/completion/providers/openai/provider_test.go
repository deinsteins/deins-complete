package openai

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/config"
)

func testConfig(baseURL string) config.AIConfig {
	return config.AIConfig{Provider: "openai-compatible", BaseURL: baseURL, APIKey: "test-key", Model: "test-model", Timeout: time.Second, MaxTokens: 128, Temperature: 0.1}
}

func testLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
func testRequest() completion.Request {
	return completion.Request{Context: completion.Context{Language: "typescript", Prefix: "const user =", Suffix: ";", FilePath: "/private/path.ts"}}
}

func TestProviderSendsCompatibleRequestAndReturnsCompletion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/chat/completions" || request.Header.Get("Authorization") != "Bearer test-key" || request.Header.Get("Content-Type") != "application/json" {
			t.Fatal("unexpected provider request")
		}
		var payload ChatCompletionRequest
		if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload.Model != "test-model" || payload.MaxTokens != 128 || payload.Temperature != 0.1 || payload.Stream || len(payload.Messages) != 2 {
			t.Fatal("unexpected payload")
		}
		if strings.Contains(payload.Messages[1].Content, "/private/path.ts") || !strings.Contains(payload.Messages[1].Content, "const user =") || !strings.Contains(payload.Messages[1].Content, "Language: typescript") {
			t.Fatal("unexpected prompt")
		}
		_, _ = writer.Write([]byte(`{"choices":[{"message":{"content":"await getUser()"},"finish_reason":"stop"}],"usage":{"prompt_tokens":10,"completion_tokens":3,"total_tokens":13}}`))
	}))
	defer server.Close()
	provider, err := New(testConfig(server.URL+"/v1/"), testLogger())
	if err != nil {
		t.Fatal(err)
	}
	result, err := provider.Complete(context.Background(), testRequest())
	if err != nil || result.Text != "await getUser()" || result.FinishReason != "stop" || result.TotalTokens != 13 {
		t.Fatalf("got %#v, %v", result, err)
	}
}

func TestProviderHandlesEmptyAndUpstreamErrors(t *testing.T) {
	cases := []struct {
		status int
		body   string
		kind   completion.ProviderErrorKind
	}{
		{http.StatusUnauthorized, `{}`, completion.ProviderAuthentication},
		{http.StatusTooManyRequests, `{}`, completion.ProviderRateLimit},
		{http.StatusInternalServerError, `{}`, completion.ProviderUnavailable},
		{http.StatusOK, `not-json`, completion.ProviderInvalidResponse},
	}
	for _, testCase := range cases {
		server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
			writer.WriteHeader(testCase.status)
			_, _ = writer.Write([]byte(testCase.body))
		}))
		provider, err := New(testConfig(server.URL), testLogger())
		if err != nil {
			t.Fatal(err)
		}
		_, err = provider.Complete(context.Background(), testRequest())
		providerError, ok := completion.AsProviderError(err)
		server.Close()
		if !ok || providerError.Kind != testCase.kind {
			t.Fatalf("status %d: got %v", testCase.status, err)
		}
	}

	emptyServer := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) { _, _ = writer.Write([]byte(`{"choices":[]}`)) }))
	defer emptyServer.Close()
	provider, _ := New(testConfig(emptyServer.URL), testLogger())
	result, err := provider.Complete(context.Background(), testRequest())
	if err != nil || result.Text != "" {
		t.Fatalf("got %#v, %v", result, err)
	}
}

func TestProviderTimeoutAndCancellation(t *testing.T) {
	timeoutServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		select {
		case <-request.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
	}))
	defer timeoutServer.Close()
	provider, _ := New(testConfig(timeoutServer.URL), testLogger())
	provider.timeout = 20 * time.Millisecond
	_, err := provider.Complete(context.Background(), testRequest())
	timeoutError, ok := completion.AsProviderError(err)
	if !ok || timeoutError.Kind != completion.ProviderTimeout {
		t.Fatalf("got %v", err)
	}

	cancellationStarted := make(chan struct{})
	cancellationServer := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		close(cancellationStarted)
		select {
		case <-request.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
	}))
	defer cancellationServer.Close()
	provider, _ = New(testConfig(cancellationServer.URL), testLogger())
	ctx, cancel := context.WithCancel(context.Background())
	result := make(chan error, 1)
	go func() { _, err := provider.Complete(ctx, testRequest()); result <- err }()
	<-cancellationStarted
	cancel()
	if err := <-result; !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}

func TestPromptAndBaseURLNormalization(t *testing.T) {
	messages := BuildMessages(testRequest())
	if len(messages) != 2 || !strings.Contains(messages[1].Content, "const user =") || !strings.Contains(messages[1].Content, ";") || strings.Contains(messages[1].Content, "test-key") || strings.Contains(messages[1].Content, "/private/path.ts") {
		t.Fatal("prompt has incorrect content")
	}
	for _, value := range []string{"https://api.example.com/v1", "https://api.example.com/v1/"} {
		baseURL, err := normalizeBaseURL(value)
		if err != nil || baseURL+"/chat/completions" != "https://api.example.com/v1/chat/completions" {
			t.Fatalf("got %q, %v", baseURL, err)
		}
	}
}
