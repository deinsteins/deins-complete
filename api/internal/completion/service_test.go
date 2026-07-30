package completion

import (
	"context"
	"testing"
)

type testProvider struct{ received context.Context }

func (provider *testProvider) Complete(ctx context.Context, _ Request) (Result, error) {
	provider.received = ctx
	return Result{Text: "ok"}, nil
}

func TestServiceForwardsRequestContext(t *testing.T) {
	provider := &testProvider{}
	service := NewService(provider)
	ctx := context.WithValue(context.Background(), "test", "value")
	result, err := service.Complete(ctx, Request{})
	if err != nil || result.Text != "ok" || provider.received != ctx {
		t.Fatalf("context was not forwarded")
	}
}
