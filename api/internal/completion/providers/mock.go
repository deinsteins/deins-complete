package providers

import (
	"context"
	"strings"
	"time"

	"deinscomplete/api/internal/completion"
)

type MockProvider struct {
	Delay time.Duration
}

func (provider MockProvider) Complete(ctx context.Context, request completion.Request) (completion.Result, error) {
	if provider.Delay > 0 {
		select {
		case <-time.After(provider.Delay):
		case <-ctx.Done():
			return completion.Result{}, ctx.Err()
		}
	}

	prefix := strings.TrimRight(request.Context.Prefix, " \t\r\n")
	switch {
	case strings.HasSuffix(prefix, "const user ="):
		return completion.Result{Text: "await getUser();"}, nil
	case strings.HasSuffix(prefix, "console."):
		return completion.Result{Text: "log()"}, nil
	default:
		return completion.Result{}, nil
	}
}
