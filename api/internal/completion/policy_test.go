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

func TestStyleInstructionAndValidation(t *testing.T) {
	style := &CodeStyle{Indentation: "spaces", IndentSize: 2, Quote: "single", Semicolons: "always"}
	if got := StyleInstruction(style); got != "indentation=spaces, indent-size=2, quotes=single, semicolons=always" {
		t.Fatalf("style instruction=%q", got)
	}
	if Validate(Request{Context: Context{Language: "typescript", Style: style}}) != nil {
		t.Fatal("valid style rejected")
	}
	if Validate(Request{Context: Context{Language: "typescript", Style: &CodeStyle{Indentation: "mixed"}}}) != ErrInvalidRequest {
		t.Fatal("invalid style accepted")
	}
}

func TestValidateSignatureHelpBounds(t *testing.T) {
	valid := Request{Context: Context{Language: "typescript"}, RepositoryContext: &RepositoryContext{SignatureHelp: &SignatureHelp{Label: "createUser(input: UserInput)", Parameter: "input: UserInput"}}}
	if Validate(valid) != nil {
		t.Fatal("valid signature help rejected")
	}
	valid.RepositoryContext.SignatureHelp.Label = ""
	if Validate(valid) != ErrInvalidRequest {
		t.Fatal("empty signature label accepted")
	}
}
