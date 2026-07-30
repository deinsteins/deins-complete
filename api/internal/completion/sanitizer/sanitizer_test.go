package sanitizer

import (
	"deinscomplete/api/internal/completion"
	"testing"
)

func clean(prefix, suffix, raw string) string {
	return New(Config{MaxLines: 3, MaxChars: 8000}).Sanitize(completion.Request{Context: completion.Context{Prefix: prefix, Suffix: suffix}}, raw)
}
func TestArtifactsAndComments(t *testing.T) {
	if got := clean("", "", "```ts\nawait getUser();\n```\nExplanation"); got != "await getUser();" {
		t.Fatal(got)
	}
	if got := clean("", "", "Here is the completion:\nawait getUser();"); got != "await getUser();" {
		t.Fatal(got)
	}
	if got := clean("", "", "// keep comment\nawait getUser();"); got != "// keep comment\nawait getUser();" {
		t.Fatal(got)
	}
}
func TestOverlapAndNewlines(t *testing.T) {
	if got := clean("const user =", "", "const user = await getUser();"); got != "await getUser();" {
		t.Fatal(got)
	}
	if got := clean("", "\r\nreturn user;", "await getUser()\r\nreturn user;"); got != "await getUser()" {
		t.Fatal(got)
	}
}
func TestLimitsAndBlankLines(t *testing.T) {
	if got := clean("", "", "a\n\n\n\nb\nline3\nline4"); got != "a\n\nb" {
		t.Fatal(got)
	}
	if got := New(Config{MaxLines: 20, MaxChars: 5}).Sanitize(completion.Request{}, "abcdef"); got != "abcde" {
		t.Fatal(got)
	}
}
