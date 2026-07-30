package router

import (
	"context"
	"errors"
	"time"

	"deinscomplete/api/internal/completion"
)

type Target struct {
	ID       string
	Provider completion.Provider
}
type Static struct {
	targets []Target
	max     int
	timeout time.Duration
}

func New(targets []Target, max int, timeout time.Duration) *Static {
	return &Static{targets: targets, max: max, timeout: timeout}
}
func (r *Static) Complete(ctx context.Context, req completion.Request) (completion.Result, error) {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	var last error
	for i, target := range r.targets {
		if i >= r.max || ctx.Err() != nil {
			break
		}
		result, err := target.Provider.Complete(ctx, req)
		if err == nil {
			return result, nil
		}
		if errors.Is(err, context.Canceled) {
			return completion.Result{}, err
		}
		last = err
		pe, ok := completion.AsProviderError(err)
		if !ok || !eligible(pe.Kind) {
			return completion.Result{}, err
		}
	}
	if ctx.Err() != nil {
		return completion.Result{}, ctx.Err()
	}
	return completion.Result{}, last
}
func (r *Static) StreamComplete(ctx context.Context, req completion.Request, onChunk func(string) error) error {
	ctx, cancel := context.WithTimeout(ctx, r.timeout)
	defer cancel()
	var last error
	for i, target := range r.targets {
		if i >= r.max || ctx.Err() != nil {
			break
		}
		emitted := false
		if stream, ok := target.Provider.(completion.StreamingProvider); ok {
			err := stream.StreamComplete(ctx, req, func(chunk string) error { emitted = emitted || chunk != ""; return onChunk(chunk) })
			if err == nil {
				return nil
			}
			if emitted || errors.Is(err, context.Canceled) {
				return err
			}
			last = err
		} else {
			result, err := target.Provider.Complete(ctx, req)
			if err == nil {
				if result.Text != "" {
					return onChunk(result.Text)
				}
				return nil
			}
			last = err
		}
		providerError, ok := completion.AsProviderError(last)
		if !ok || !eligible(providerError.Kind) {
			return last
		}
	}
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return last
}
func eligible(k completion.ProviderErrorKind) bool {
	return k == completion.ProviderAuthentication || k == completion.ProviderRateLimit || k == completion.ProviderTimeout || k == completion.ProviderUnavailable || k == completion.ProviderInvalidResponse
}

var _ completion.Provider = (*Static)(nil)
