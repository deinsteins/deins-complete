package completion

import "testing"

func TestIntentPolicyBoundsGeneration(t *testing.T) {
	if !ValidIntent("member-access") || ValidIntent("unknown") {
		t.Fatal("intent validation mismatch")
	}
	if got := MaxTokensForIntent("member-access", 256); got != 48 {
		t.Fatalf("member tokens=%d", got)
	}
	if got := MaxTokensForIntent("function-body", 256); got != 256 {
		t.Fatalf("function body tokens=%d", got)
	}
	if got := MaxLinesForIntent("import", 20); got != 1 {
		t.Fatalf("import lines=%d", got)
	}
}

func TestValidateRejectsUnknownIntent(t *testing.T) {
	request := Request{Context: Context{Language: "typescript"}, Intent: "unknown"}
	if Validate(request) != ErrInvalidRequest {
		t.Fatal("expected invalid intent rejection")
	}
}
