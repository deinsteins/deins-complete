package auth

import "testing"

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
