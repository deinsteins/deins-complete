package anthropic

import (
	"context"
	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/config"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func cfg(url string) config.AIConfig {
	return config.AIConfig{Provider: "anthropic", Anthropic: config.AnthropicConfig{BaseURL: url, APIKey: "test-key", Model: "claude", Version: "2023-06-01"}, Timeout: time.Second, MaxTokens: 128, Temperature: .1}
}
func logger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }
func req() completion.Request {
	return completion.Request{Context: completion.Context{Language: "typescript", Prefix: "const user =", Suffix: ";", FilePath: "/secret/file.ts"}}
}

func TestProviderBlocksAndHeaders(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" || r.Header.Get("x-api-key") != "test-key" || r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Fatal("bad request")
		}
		var b requestBody
		_ = json.NewDecoder(r.Body).Decode(&b)
		if b.Model != "claude" || strings.Contains(b.Messages[0].Content, "/secret/file.ts") {
			t.Fatal("bad body")
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"tool_use"},{"type":"text","text":"await "},{"type":"text","text":"getUser()"}],"stop_reason":"end_turn","usage":{"input_tokens":10,"output_tokens":3}}`))
	}))
	defer s.Close()
	p, _ := New(cfg(s.URL+"/"), logger())
	result, err := p.Complete(context.Background(), req())
	if err != nil || result.Text != "await getUser()" || result.TotalTokens != 13 {
		t.Fatalf("%#v %v", result, err)
	}
}

func TestProviderErrorsTimeoutAndCancellation(t *testing.T) {
	for _, tc := range []struct {
		status int
		kind   completion.ProviderErrorKind
	}{{401, completion.ProviderAuthentication}, {429, completion.ProviderRateLimit}, {500, completion.ProviderUnavailable}} {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(tc.status) }))
		p, _ := New(cfg(s.URL), logger())
		_, err := p.Complete(context.Background(), req())
		got, ok := completion.AsProviderError(err)
		s.Close()
		if !ok || got.Kind != tc.kind {
			t.Fatal(err)
		}
	}
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(100 * time.Millisecond):
		}
	}))
	defer s.Close()
	p, _ := New(cfg(s.URL), logger())
	p.timeout = 20 * time.Millisecond
	_, err := p.Complete(context.Background(), req())
	got, ok := completion.AsProviderError(err)
	if !ok || got.Kind != completion.ProviderTimeout {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = p.Complete(ctx, req())
	if !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestURLNormalization(t *testing.T) {
	for _, v := range []string{"https://api.anthropic.com", "https://api.anthropic.com/", "https://api.anthropic.com/v1"} {
		u, e := normalizeBaseURL(v)
		if e != nil || u+"/v1/messages" != "https://api.anthropic.com/v1/messages" {
			t.Fatal(v)
		}
	}
}

func TestIntentGuidesPromptAndBoundsOutputTokens(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body requestBody
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body.MaxTokens != 48 || !strings.Contains(body.Messages[0].Content, "Completion intent: member-access") {
			t.Fatalf("intent policy missing: %#v", body)
		}
		_, _ = w.Write([]byte(`{"content":[{"type":"text","text":"name"}]}`))
	}))
	defer s.Close()
	p, err := New(cfg(s.URL), logger())
	if err != nil {
		t.Fatal(err)
	}
	request := req()
	request.Intent = "member-access"
	if _, err := p.Complete(context.Background(), request); err != nil {
		t.Fatal(err)
	}
}
