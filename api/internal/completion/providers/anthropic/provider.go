package anthropic

import (
	"bytes"
	"context"
	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/config"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type requestBody struct {
	Model       string    `json:"model"`
	MaxTokens   int       `json:"max_tokens"`
	Temperature float64   `json:"temperature"`
	System      string    `json:"system"`
	Messages    []message `json:"messages"`
}
type responseBody struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}
type Provider struct {
	baseURL, apiKey, model, version string
	timeout                         time.Duration
	maxTokens                       int
	temperature                     float64
	client                          *http.Client
	logger                          *slog.Logger
}

func New(cfg config.AIConfig, logger *slog.Logger) (*Provider, error) {
	baseURL, err := normalizeBaseURL(cfg.Anthropic.BaseURL)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 20
	transport.IdleConnTimeout = 90 * time.Second
	return &Provider{baseURL: baseURL, apiKey: cfg.Anthropic.APIKey, model: cfg.Anthropic.Model, version: cfg.Anthropic.Version, timeout: cfg.Timeout, maxTokens: cfg.MaxTokens, temperature: cfg.Temperature, client: &http.Client{Transport: transport}, logger: logger}, nil
}

func (p *Provider) Complete(ctx context.Context, req completion.Request) (completion.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()
	body, _ := json.Marshal(requestBody{Model: p.model, MaxTokens: p.maxTokens, Temperature: p.temperature, System: systemPrompt, Messages: []message{{Role: "user", Content: userPrompt(req)}}})
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, p.baseURL+"/v1/messages", bytes.NewReader(body))
	if err != nil {
		return completion.Result{}, completion.NewProviderError(completion.ProviderInvalidResponse, err)
	}
	r.Header.Set("x-api-key", p.apiKey)
	r.Header.Set("anthropic-version", p.version)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json")
	res, err := p.client.Do(r)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return completion.Result{}, completion.NewProviderError(completion.ProviderTimeout, err)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return completion.Result{}, ctx.Err()
		}
		return completion.Result{}, completion.NewProviderError(completion.ProviderUnavailable, err)
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, io.LimitReader(res.Body, 4096))
		return completion.Result{}, statusError(res.StatusCode)
	}
	var payload responseBody
	if err := json.NewDecoder(io.LimitReader(res.Body, 1<<20)).Decode(&payload); err != nil {
		return completion.Result{}, completion.NewProviderError(completion.ProviderInvalidResponse, err)
	}
	var parts []string
	for _, block := range payload.Content {
		if block.Type == "text" {
			parts = append(parts, block.Text)
		}
	}
	return completion.Result{Text: strings.Join(parts, ""), FinishReason: payload.StopReason, PromptTokens: payload.Usage.InputTokens, CompletionTokens: payload.Usage.OutputTokens, TotalTokens: payload.Usage.InputTokens + payload.Usage.OutputTokens}, nil
}

const systemPrompt = "You are a code completion engine. Return only code inserted at the cursor. Do not explain, use Markdown, or repeat surrounding code. Prefer the smallest useful completion."

func userPrompt(r completion.Request) string {
	prompt := "Language: " + r.Context.Language + "\n\n<PREFIX>\n" + r.Context.Prefix + "\n</PREFIX>\n\n<SUFFIX>\n" + r.Context.Suffix + "\n</SUFFIX>"
	if r.RepositoryContext != nil && (len(r.RepositoryContext.Files) > 0 || len(r.RepositoryContext.Dependencies) > 0 || len(r.RepositoryContext.Symbols) > 0 || r.RepositoryContext.Focus != "") {
		prompt += "\n\n<REPOSITORY_CONTEXT>"
		if r.RepositoryContext.Focus != "" {
			prompt += "\n<COMPLETION_FOCUS>" + r.RepositoryContext.Focus + "</COMPLETION_FOCUS>"
		}
		if len(r.RepositoryContext.Dependencies) > 0 {
			prompt += "\n<DEPENDENCIES>" + strings.Join(r.RepositoryContext.Dependencies, ", ") + "</DEPENDENCIES>"
		}
		for _, file := range r.RepositoryContext.Files {
			prompt += "\n<FILE path=\"" + file.Path + "\" language=\"" + file.Language + "\">\n" + file.Content + "\n</FILE>"
		}
		for _, symbol := range r.RepositoryContext.Symbols {
			prompt += "\n<SYMBOL name=\"" + symbol.Name + "\" kind=\"" + symbol.Kind + "\" source=\"" + symbol.FilePath + "\">" + symbol.Signature + "</SYMBOL>"
		}
		prompt += "\n</REPOSITORY_CONTEXT>"
	}
	return prompt + "\n\nReturn only the code inserted between PREFIX and SUFFIX."
}
func normalizeBaseURL(v string) (string, error) {
	u, e := url.Parse(v)
	if e != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("invalid Anthropic base URL")
	}
	return strings.TrimSuffix(strings.TrimRight(u.String(), "/"), "/v1"), nil
}
func statusError(s int) error {
	switch {
	case s == 401 || s == 403:
		return completion.NewProviderError(completion.ProviderAuthentication, errors.New("upstream authentication failed"))
	case s == 429:
		return completion.NewProviderError(completion.ProviderRateLimit, errors.New("upstream rate limited"))
	case s == 408:
		return completion.NewProviderError(completion.ProviderTimeout, errors.New("upstream timeout"))
	default:
		return completion.NewProviderError(completion.ProviderUnavailable, errors.New("upstream unavailable"))
	}
}
