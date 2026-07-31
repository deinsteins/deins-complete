package accountauth

import (
	"strings"
	"testing"
	"time"
)

func TestAccessTokenAndOpaqueHash(t *testing.T) {
	service := New("01234567890123456789012345678901", time.Hour)
	token, err := service.IssueAccessToken("user-1")
	if err != nil {
		t.Fatal(err)
	}
	if id, err := service.ValidateAccessToken(token); err != nil || id != "user-1" {
		t.Fatalf("id=%q err=%v", id, err)
	}
	if _, err := service.ValidateAccessToken(token + "x"); err == nil {
		t.Fatal("expected tamper rejection")
	}
	opaque, err := NewOpaqueToken()
	if err != nil || len(opaque) < 32 {
		t.Fatalf("opaque=%q err=%v", opaque, err)
	}
	if strings.Contains(HashToken(opaque), opaque) {
		t.Fatal("hash leaked token")
	}
}
