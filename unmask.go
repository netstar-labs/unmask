package unmask

import (
	"sort"
	"strings"
)

// Report is the UTS-39 confusable analysis of a label.
type Report struct {
	// Skeleton is the confusable-skeleton clustering key. It is MANY-TO-ONE (many
	// look-alikes share one skeleton) and MUST NOT be used as a lookup key or hash
	// input — see [Confusable].
	Skeleton string
	// Scripts are the distinct Unicode scripts present in the label, excluding the
	// script-neutral Common, Inherited, and Unknown, sorted. One entry is single-script.
	Scripts []string
	// MixedScript reports whether the label mixes two or more scripts — the
	// strongest single homograph signal, and independent of any target list.
	MixedScript bool
}

// Analyze returns the confusable [Report] for label. The caller supplies the
// U-label (the pre-punycode Unicode host, e.g. from normie's Display / idna.ToUnicode)
// and normalises case first; Analyze does not lower-case or decode punycode.
func Analyze(label string) Report {
	return Report{
		Skeleton:    Skeleton(label),
		Scripts:     Scripts(label),
		MixedScript: MixedScript(label),
	}
}

// Skeleton returns the UTS-39 confusable skeleton of s: each rune is replaced by
// its confusable prototype (confusables.txt MA), collapsing look-alikes onto one
// form (`pаypаl` with Cyrillic а → `paypal`). The result is lower-cased so the
// index is case-insensitive — several prototypes are upper-case (digit 0 → "O"),
// so without folding `g00gle` would skeletonise to "gOOgle" and fail to JOIN a
// lower-case brand. The skeleton is a clustering index, never a lookup key.
//
// s must be a U-label the caller has already normalised to NFC (as idna.ToUnicode
// yields); the skeleton does not fold combining marks (see the package doc).
func Skeleton(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if p, ok := confusable[r]; ok {
			b.WriteString(p)
		} else {
			b.WriteRune(r)
		}
	}
	return strings.ToLower(b.String())
}

// Confusable reports whether a and b are distinct identities that skeletonise to
// the same form — the correct UTS-39 detection test. Use it as a JOIN against a
// brand target list (Confusable(candidate, brand)); never as skeleton == skeleton
// alone, which would flag the legitimate brand against its own homograph.
//
// "Distinct identities" is case-fold-insensitive, not byte-exact: DNS names are
// case-insensitive, so "PayPal.com" and "paypal.com" are the same identity, not
// two different ones. A byte-exact guard would let that pair through (both are
// pure ASCII, so they trivially skeletonise to the same lower-cased form) and
// report the brand's own domain as confusable with itself the moment it appears
// in a different case anywhere in the pipeline (a human-typed brand list vs. a
// lower-cased candidate stream, say) — exactly the false positive this guard
// exists to prevent, just arriving through the case channel instead of the
// Unicode-homograph channel.
func Confusable(a, b string) bool {
	return !strings.EqualFold(a, b) && Skeleton(a) == Skeleton(b)
}

// neutral reports whether a script is script-neutral (Common or Inherited: digits,
// punctuation, combining marks; or Unknown: unassigned code points and gaps between
// ranges, per scriptOf's fallback) — excluded from the mixed-script signal. An
// unassigned code point is not a real script, so counting it as one would make an
// unassigned or private-use rune alongside a single real script register as
// "mixed" — a false-positive direction rather than a genuine cross-script signal.
func neutral(script string) bool {
	return script == "Common" || script == "Inherited" || script == "Unknown"
}

// Scripts returns the distinct scripts present in s, excluding the script-neutral
// Common and Inherited (digits, punctuation, combining marks) and Unknown
// (unassigned code points and gaps between ranges), sorted.
func Scripts(s string) []string {
	set := make(map[string]struct{})
	for _, r := range s {
		if sc := scriptOf(r); !neutral(sc) {
			set[sc] = struct{}{}
		}
	}
	out := make([]string, 0, len(set))
	for sc := range set {
		out = append(out, sc)
	}
	sort.Strings(out)
	return out
}

// MixedScript reports whether s mixes two or more scripts (ignoring the neutral
// Common, Inherited, and Unknown). A Latin label with a single Cyrillic look-alike is mixed;
// an all-Cyrillic label that reads as Latin is NOT (that is a whole-script
// confusable, caught by [Skeleton], not this).
func MixedScript(s string) bool {
	first := ""
	for _, r := range s {
		sc := scriptOf(r)
		if neutral(sc) {
			continue
		}
		if first == "" {
			first = sc
		} else if sc != first {
			return true
		}
	}
	return false
}

// Unicode reports the Unicode version the confusable + script tables were
// generated from. Unlike a UTS-46 A-label mapping, the skeleton is not a lookup
// key, so this bumps freely with each release — it is informational, not a
// migration stamp.
func Unicode() string { return unicodeVersion }

// Map returns the confusable prototype of r — the rune(s) r collapses to in the
// skeleton — and whether r has a mapping. A prototype can be several runes (a
// multi-rune confusable such as m → "rn"), so it is returned as a []rune; an unmapped
// rune returns (nil, false) and skeletonises to itself. It exposes the raw per-rune
// confusables table for callers that need the mapping directly — a confusability-weighted
// substitution cost, a look-alike generator — without re-parsing the Unicode data.
func Map(r rune) ([]rune, bool) {
	if p, ok := confusable[r]; ok {
		return []rune(p), true
	}
	return nil, false
}

// scriptOf returns the Unicode Script property of r via binary search over the
// generated ranges (sorted by lo and non-overlapping, so also sorted by hi).
// Unassigned/unlisted code points — including gaps between ranges — report
// "Unknown".
func scriptOf(r rune) string {
	i := sort.Search(len(scriptRanges), func(i int) bool { return scriptRanges[i].hi >= r })
	if i < len(scriptRanges) && scriptRanges[i].lo <= r {
		return scriptRanges[i].script
	}
	return "Unknown"
}
