package unmask

import (
	"strings"
	"testing"
)

// TestMapCyrillic checks the headline case: Cyrillic а maps to Latin a.
func TestMapCyrillic(t *testing.T) {
	got, ok := Map('а') // U+0430 CYRILLIC SMALL LETTER A
	if !ok || len(got) == 0 {
		t.Fatalf("Map(Cyrillic а) = (%q, %v), want a non-empty prototype", string(got), ok)
	}
	if strings.ToLower(string(got)) != "a" {
		t.Errorf("Map(Cyrillic а) = %q, want prototype 'a'", string(got))
	}
}

// TestMapConsistentWithSkeleton is the invariant that ties Map to the table Skeleton
// uses: for any rune, mapping it through Map (else leaving it) and lower-casing must
// equal Skeleton of that single rune. Holds regardless of which runes the table maps.
func TestMapConsistentWithSkeleton(t *testing.T) {
	for _, r := range []rune{'а', 'a', '0', '1', 'm', 'q', 'λ', '中', 'ο'} {
		p, ok := Map(r)
		want := strings.ToLower(string(r))
		if ok {
			want = strings.ToLower(string(p))
		}
		if got := Skeleton(string(r)); got != want {
			t.Errorf("Map(%q)=(%q,%v) inconsistent with Skeleton=%q (want %q)", r, string(p), ok, got, want)
		}
	}
}
