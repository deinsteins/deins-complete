package completion

import (
	"context"
)

type Service struct {
	provider Provider
}

type Provider interface {
	Complete(ctx context.Context, request Request) (Result, error)
}

func NewService(provider Provider) *Service {
	return &Service{provider: provider}
}

func (service *Service) Complete(ctx context.Context, request Request) (Result, error) {
	return service.provider.Complete(ctx, request)
}
