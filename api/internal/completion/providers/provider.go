package providers

import (
	"context"

	"deinscomplete/api/internal/completion"
)

type Provider interface {
	Complete(ctx context.Context, request completion.Request) (completion.Result, error)
}
