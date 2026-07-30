package fim

import (
	"deinscomplete/api/internal/completion"
	"testing"
)

func TestFormatAndStripEnd(t *testing.T) {
	got := Format(completion.Request{Context: completion.Context{Prefix: "return ", Suffix: ";\n}"}}, Config{"<P>", "<S>", "<M>", "<END>"})
	if got != "<P>return <S>;\n}<M>" {
		t.Fatal(got)
	}
	if StripEnd("value<END>", "<END>") != "value" {
		t.Fatal("end token leaked")
	}
}
