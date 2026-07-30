package completion

import (
	"context"
)

type Service struct {
	provider  Provider
	sanitizer Sanitizer
}

type Provider interface {
	Complete(ctx context.Context, request Request) (Result, error)
}
type StreamingProvider interface {
	StreamComplete(ctx context.Context, request Request, onChunk func(string) error) error
}
type Sanitizer interface{ Sanitize(Request, string) string }
type identitySanitizer struct{}

func (identitySanitizer) Sanitize(_ Request, text string) string { return text }

func NewService(provider Provider, sanitizers ...Sanitizer) *Service {
	var cleaner Sanitizer = identitySanitizer{}
	if len(sanitizers) > 0 {
		cleaner = sanitizers[0]
	}
	return &Service{provider: provider, sanitizer: cleaner}
}

func (service *Service) Complete(ctx context.Context, request Request) (Result, error) {
	result, err := service.provider.Complete(ctx, request)
	if err != nil {
		return Result{}, err
	}
	result.Text = service.sanitizer.Sanitize(request, result.Text)
	return result, nil
}

func (service *Service) Stream(ctx context.Context, request Request, onChunk func(string) error) (Result, error) {
	stream, ok := service.provider.(StreamingProvider)
	if !ok {
		return service.Complete(ctx, request)
	}
	var raw string
	err := stream.StreamComplete(ctx, request, func(chunk string) error { raw += chunk; return onChunk(chunk) })
	if err != nil {
		return Result{}, err
	}
	return Result{Text: service.sanitizer.Sanitize(request, raw)}, nil
}
