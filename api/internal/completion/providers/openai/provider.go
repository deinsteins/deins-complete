package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"deinscomplete/api/internal/completion"
	"deinscomplete/api/internal/config"
)

const maxProviderResponseBytes = 1 << 20

type Provider struct {
	baseURL     string
	apiKey      string
	model       string
	timeout     time.Duration
	maxTokens   int
	temperature float64
	client      *http.Client
	logger      *slog.Logger
}

func New(configuration config.AIConfig, logger *slog.Logger) (*Provider, error) {
	baseURL, err := normalizeBaseURL(configuration.BaseURL)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 20
	transport.IdleConnTimeout = 90 * time.Second
	return &Provider{
		baseURL: baseURL, apiKey: configuration.APIKey, model: configuration.Model,
		timeout: configuration.Timeout, maxTokens: configuration.MaxTokens, temperature: configuration.Temperature,
		client: &http.Client{Transport: transport}, logger: logger,
	}, nil
}

func (provider *Provider) Complete(ctx context.Context, request completion.Request) (completion.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, provider.timeout)
	defer cancel()
	startedAt := time.Now()
	body, err := json.Marshal(ChatCompletionRequest{
		Model: provider.model, Messages: BuildMessages(request), Temperature: provider.temperature, MaxTokens: provider.maxTokens, Stream: false,
	})
	if err != nil {
		return completion.Result{}, completion.NewProviderError(completion.ProviderInvalidResponse, err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return completion.Result{}, completion.NewProviderError(completion.ProviderInvalidResponse, err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+provider.apiKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	httpRequest.Header.Set("Accept", "application/json")
	response, err := provider.client.Do(httpRequest)
	if err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return completion.Result{}, completion.NewProviderError(completion.ProviderTimeout, err)
		}
		if errors.Is(ctx.Err(), context.Canceled) {
			return completion.Result{}, ctx.Err()
		}
		return completion.Result{}, completion.NewProviderError(completion.ProviderUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, 4096))
		provider.logger.Warn("provider request failed", "provider", "openai-compatible", "model", provider.model, "status", response.StatusCode, "duration_ms", time.Since(startedAt).Milliseconds())
		return completion.Result{}, statusError(response.StatusCode)
	}
	var payload chatCompletionResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, maxProviderResponseBytes)).Decode(&payload); err != nil {
		return completion.Result{}, completion.NewProviderError(completion.ProviderInvalidResponse, err)
	}
	if len(payload.Choices) == 0 || strings.TrimSpace(payload.Choices[0].Message.Content) == "" {
		return completion.Result{}, nil
	}
	result := completion.Result{Text: payload.Choices[0].Message.Content, FinishReason: payload.Choices[0].FinishReason}
	if payload.Usage != nil {
		result.PromptTokens = payload.Usage.PromptTokens
		result.CompletionTokens = payload.Usage.CompletionTokens
		result.TotalTokens = payload.Usage.TotalTokens
	}
	provider.logger.Debug("provider request completed", "provider", "openai-compatible", "model", provider.model, "duration_ms", time.Since(startedAt).Milliseconds())
	return result, nil
}

func normalizeBaseURL(value string) (string, error) {
	parsedURL, err := url.Parse(value)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Host == "" || parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return "", errors.New("invalid OpenAI-compatible base URL")
	}
	return strings.TrimRight(parsedURL.String(), "/"), nil
}

func statusError(status int) error {
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		return completion.NewProviderError(completion.ProviderAuthentication, errors.New("upstream authentication failed"))
	case status == http.StatusTooManyRequests:
		return completion.NewProviderError(completion.ProviderRateLimit, errors.New("upstream rate limited"))
	case status == http.StatusRequestTimeout:
		return completion.NewProviderError(completion.ProviderTimeout, errors.New("upstream timeout"))
	case status >= http.StatusInternalServerError:
		return completion.NewProviderError(completion.ProviderUnavailable, errors.New("upstream unavailable"))
	default:
		return completion.NewProviderError(completion.ProviderUnavailable, errors.New("upstream request failed"))
	}
}
