package openai

import (
	"bufio"
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
	"deinscomplete/api/internal/completion/fim"
	"deinscomplete/api/internal/config"
)

const maxProviderResponseBytes = 1 << 20

type Provider struct {
	baseURL                 string
	apiKey                  string
	model                   string
	timeout                 time.Duration
	maxTokens               int
	temperature             float64
	apiMode, completionMode string
	fim                     fim.Config
	client                  *http.Client
	logger                  *slog.Logger
}

func New(configuration config.AIConfig, logger *slog.Logger) (*Provider, error) {
	baseURL, err := normalizeBaseURL(configuration.OpenAI.BaseURL)
	if err != nil {
		return nil, err
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = 100
	transport.MaxIdleConnsPerHost = 20
	transport.IdleConnTimeout = 90 * time.Second
	return &Provider{
		baseURL: baseURL, apiKey: configuration.OpenAI.APIKey, model: configuration.OpenAI.Model,
		timeout: configuration.Timeout, maxTokens: configuration.MaxTokens, temperature: configuration.Temperature,
		apiMode: configuration.APIMode, completionMode: configuration.CompletionMode, fim: fim.Config{PrefixToken: configuration.FIMPrefixToken, SuffixToken: configuration.FIMSuffixToken, MiddleToken: configuration.FIMMiddleToken, EndToken: configuration.FIMEndToken},
		client: &http.Client{Transport: transport}, logger: logger,
	}, nil
}

func (provider *Provider) Complete(ctx context.Context, request completion.Request) (completion.Result, error) {
	if provider.apiMode == "completion" {
		return provider.completeRaw(ctx, request)
	}
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

// StreamComplete consumes the OpenAI-compatible SSE format used by MiMo.
func (provider *Provider) StreamComplete(ctx context.Context, request completion.Request, onChunk func(string) error) error {
	if provider.apiMode != "chat" {
		return completion.NewProviderError(completion.ProviderUnavailable, errors.New("streaming requires chat API mode"))
	}
	ctx, cancel := context.WithTimeout(ctx, provider.timeout)
	defer cancel()
	body, err := json.Marshal(ChatCompletionRequest{Model: provider.model, Messages: BuildMessages(request), Temperature: provider.temperature, MaxTokens: provider.maxTokens, Stream: true})
	if err != nil {
		return completion.NewProviderError(completion.ProviderInvalidResponse, err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return completion.NewProviderError(completion.ProviderInvalidResponse, err)
	}
	req.Header.Set("Authorization", "Bearer "+provider.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	response, err := provider.client.Do(req)
	if err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return ctx.Err()
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return completion.NewProviderError(completion.ProviderTimeout, err)
		}
		return completion.NewProviderError(completion.ProviderUnavailable, err)
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return statusError(response.StatusCode)
	}
	scanner := bufio.NewScanner(io.LimitReader(response.Body, maxProviderResponseBytes))
	scanner.Buffer(make([]byte, 4096), maxProviderResponseBytes)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			return nil
		}
		var event struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return completion.NewProviderError(completion.ProviderInvalidResponse, err)
		}
		for _, choice := range event.Choices {
			if choice.Delta.Content != "" {
				if err := onChunk(choice.Delta.Content); err != nil {
					return err
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(ctx.Err(), context.Canceled) {
			return ctx.Err()
		}
		return completion.NewProviderError(completion.ProviderInvalidResponse, err)
	}
	return nil
}

func (provider *Provider) completeRaw(ctx context.Context, request completion.Request) (completion.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, provider.timeout)
	defer cancel()
	prompt := fim.Format(request, provider.fim)
	body, err := json.Marshal(CompletionRequest{Model: provider.model, Prompt: prompt, Temperature: provider.temperature, MaxTokens: provider.maxTokens})
	if err != nil {
		return completion.Result{}, completion.NewProviderError(completion.ProviderInvalidResponse, err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, provider.baseURL+"/completions", bytes.NewReader(body))
	if err != nil {
		return completion.Result{}, completion.NewProviderError(completion.ProviderInvalidResponse, err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+provider.apiKey)
	httpRequest.Header.Set("Content-Type", "application/json")
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
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return completion.Result{}, statusError(response.StatusCode)
	}
	var payload completionResponse
	if err := json.NewDecoder(io.LimitReader(response.Body, maxProviderResponseBytes)).Decode(&payload); err != nil {
		return completion.Result{}, completion.NewProviderError(completion.ProviderInvalidResponse, err)
	}
	if len(payload.Choices) == 0 {
		return completion.Result{}, nil
	}
	return completion.Result{Text: fim.StripEnd(payload.Choices[0].Text, provider.fim.EndToken), FinishReason: payload.Choices[0].FinishReason}, nil
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
