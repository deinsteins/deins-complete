package completion

import (
	"context"
	"errors"
	"strings"
)

var errUsefulLineComplete = errors.New("useful completion line complete")

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
	singleLine := singleLineFocus(request)
	err := stream.StreamComplete(ctx, request, func(chunk string) error {
		previousLength := len(raw)
		raw += chunk
		stop := false
		if singleLine {
			if newline := strings.Index(raw, "\n"); newline >= 0 {
				chunk = chunk[:maxInt(0, newline-previousLength)]
				raw = raw[:newline]
				stop = true
			}
		}
		if chunk != "" {
			if err := onChunk(chunk); err != nil {
				return err
			}
		}
		if stop {
			return errUsefulLineComplete
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		return nil
	})
	if err != nil && !errors.Is(err, errUsefulLineComplete) {
		return Result{}, err
	}
	return Result{Text: service.sanitizer.Sanitize(request, raw)}, nil
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func singleLineFocus(request Request) bool {
	if request.Intent != "" {
		return SingleLineIntent(request.Intent)
	}
	if request.RepositoryContext == nil {
		return false
	}
	switch request.RepositoryContext.Focus {
	case "component-props", "member-access", "function-arguments", "tailwind-class":
		return true
	default:
		return false
	}
}
