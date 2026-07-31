package auth

import (
	"strings"
	"testing"
)

func TestTokenValidatesSignatureAndVersion(t *testing.T) {
	service := New("01234567890123456789012345678901", 1, 0)
	token, err := service.Issue("installation-1")
	if err != nil {
		t.Fatal(err)
	}
	if id, err := service.Validate(token); err != nil || id != "installation-1" {
		t.Fatalf("unexpected token validation: %q %v", id, err)
	}
	replacement := "x"
	if token[len(token)-1:] == replacement {
		replacement = "y"
	}
	if _, err := service.Validate(token[:len(token)-1] + replacement); err == nil {
		t.Fatal("expected tampered token rejection")
	}
	if _, err := New("01234567890123456789012345678901", 2, 0).Validate(token); err == nil {
		t.Fatal("expected version rejection")
	}
}

func TestTokenRejectsNonCanonicalSignatureEncoding(t *testing.T) {
	service := New("01234567890123456789012345678901", 1, 0)
	token, err := service.Issue("installation-1")
	if err != nil {
		t.Fatal(err)
	}
	separator := len(token) - 43
	signature := token[separator:]
	last := signature[len(signature)-1]
	alphabet := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"
	index := strings.IndexByte(alphabet, last)
	alias := alphabet[(index&^3)|((index+1)&3)]
	if alias == last {
		alias = alphabet[(index&^3)|((index+2)&3)]
	}
	if _, err := service.Validate(token[:len(token)-1] + string(alias)); err == nil {
		t.Fatal("expected non-canonical token rejection")
	}
}
