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

type streamingTestProvider struct{ testProvider }

func (provider *streamingTestProvider) StreamComplete(_ context.Context, _ Request, onChunk func(string) error) error {
	if err := onChunk("first line\nsecond line"); err != nil {
		return err
	}
	return nil
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

func TestStreamStopsAfterUsefulSingleLine(t *testing.T) {
	service := NewService(&streamingTestProvider{})
	request := Request{Intent: "member-access"}
	var streamed string
	result, err := service.Stream(context.Background(), request, func(chunk string) error { streamed += chunk; return nil })
	if err != nil || result.Text != "first line" || streamed != "first line" {
		t.Fatalf("text=%q streamed=%q err=%v", result.Text, streamed, err)
	}
}
