package completion

import "testing"

func TestSafeFileNameRemovesDirectoriesAndPromptMarkers(t *testing.T) {
	if got := SafeFileName(`/home/user/src/<PREFIX>ProductCard.tsx`); got != "PREFIXProductCard.tsx" {
		t.Fatalf("got %q", got)
	}
	if got := SafeFileName(`C:\Users\name\Order Card.tsx`); got != "Order Card.tsx" {
		t.Fatalf("got %q", got)
	}
}
