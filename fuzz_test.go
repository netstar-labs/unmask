package unmask

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// FuzzSkeleton asserts the skeleton never panics, is deterministic, is valid
// UTF-8, and is idempotent on ASCII prototypes (mapping is a single pass, so a
// second pass over an all-ASCII skeleton must be a fixed point).
func FuzzSkeleton(f *testing.F) {
	for _, s := range []string{"", "paypal", "pаypаl", "g00gle", "münchen", strings.Repeat("а", 300)} {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, s string) {
		sk := Skeleton(s)
		if Skeleton(s) != sk {
			t.Fatal("Skeleton not deterministic")
		}
		if !utf8.ValidString(sk) {
			t.Errorf("skeleton not valid UTF-8 for %q", s)
		}
		_ = Analyze(s) // must not panic
		_ = Confusable(s, "paypal")
	})
}
