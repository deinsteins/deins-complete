package fim

import (
	"deinscomplete/api/internal/completion"
	"testing"
)

func TestFormatAndStripEnd(t *testing.T) {
	got := Format(completion.Request{Context: completion.Context{Prefix: "return ", Suffix: ";\n}"}}, Config{PrefixToken: "<P>", SuffixToken: "<S>", MiddleToken: "<M>", EndToken: "<END>"})
	if got != "<P>return <S>;\n}<M>" {
		t.Fatal(got)
	}
	if StripEnd("value<END>", "<END>") != "value" {
		t.Fatal("end token leaked")
	}
}

func TestFormatOptionallyIncludesSafeFilename(t *testing.T) {
	request := completion.Request{Context: completion.Context{
		Prefix: "export function ProductCard()", Suffix: "{}", Language: "typescriptreact",
		FilePath: "src/components/ProductCard.tsx",
	}}
	config := Config{PrefixToken: "<P>", SuffixToken: "<S>", MiddleToken: "<M>", FilenameContext: true}
	got := Format(request, config)
	want := "<P>// File: ProductCard.tsx\nexport function ProductCard()<S>{}<M>"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
