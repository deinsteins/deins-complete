package providers

import (
	"context"
	"errors"
	"testing"
	"time"

	"deinscomplete/api/internal/completion"
)

func TestMockProviderRules(t *testing.T) {
	provider := MockProvider{}
	result, err := provider.Complete(context.Background(), completion.Request{Context: completion.Context{Prefix: "const user =", Language: "typescript"}})
	if err != nil || result.Text != "await getUser();" {
		t.Fatalf("got %#v, %v", result, err)
	}
}

func TestMockProviderHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	provider := MockProvider{Delay: time.Second}
	cancel()
	_, err := provider.Complete(ctx, completion.Request{})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
}
