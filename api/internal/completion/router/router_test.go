package router

import (
	"context"
	"deinscomplete/api/internal/completion"
	"errors"
	"testing"
	"time"
)

type fake struct {
	result completion.Result
	err    error
	calls  int
}

func (f *fake) Complete(context.Context, completion.Request) (completion.Result, error) {
	f.calls++
	return f.result, f.err
}
func TestFallbackAndEmptySuccess(t *testing.T) {
	primary := &fake{err: completion.NewProviderError(completion.ProviderTimeout, errors.New("timeout"))}
	fallback := &fake{result: completion.Result{Text: "ok"}}
	r := New([]Target{{"primary", primary}, {"fallback", fallback}}, 2, time.Second)
	got, err := r.Complete(context.Background(), completion.Request{})
	if err != nil || got.Text != "ok" || fallback.calls != 1 {
		t.Fatal("fallback failed")
	}
	empty := &fake{}
	backup := &fake{result: completion.Result{Text: "wrong"}}
	got, err = New([]Target{{"primary", empty}, {"backup", backup}}, 2, time.Second).Complete(context.Background(), completion.Request{})
	if err != nil || got.Text != "" || backup.calls != 0 {
		t.Fatal("empty must succeed")
	}
}
func TestCancellationDoesNotFallback(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	primary := &fake{err: context.Canceled}
	backup := &fake{}
	_, _ = New([]Target{{"p", primary}, {"b", backup}}, 2, time.Second).Complete(ctx, completion.Request{})
	if backup.calls != 0 {
		t.Fatal("cancelled request fell back")
	}
}
