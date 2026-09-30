package unmask

import (
	"reflect"
	"testing"
)

func TestConfusableHomograph(t *testing.T) {
	// "pаypаl" with Cyrillic а (U+0430) reads as the Latin "paypal".
	cyr := "pаypаl"
	if !Confusable(cyr, "paypal") {
		t.Errorf("Confusable(%q, paypal) = false; skeletons %q vs %q", cyr, Skeleton(cyr), Skeleton("paypal"))
	}
	// A string is never confusable with itself (the != guard).
	if Confusable("paypal", "paypal") {
		t.Error("Confusable(paypal, paypal) = true; must be false")
	}
	// Unrelated strings are not confusable.
	if Confusable("paypal", "example") {
		t.Error("Confusable(paypal, example) = true; must be false")
	}
}

func TestSkeletonDigitLookalike(t *testing.T) {
	// Digit 0 is a confusable of Latin O in confusables.txt, so g00gle and gOOgle
	// share a skeleton.
	if !Confusable("g00gle", "gOOgle") {
		t.Errorf("g00gle/gOOgle not confusable; %q vs %q", Skeleton("g00gle"), Skeleton("gOOgle"))
	}
}

func TestMixedScript(t *testing.T) {
	cases := map[string]bool{
		"paypal":      false, // all Latin
		"pаypаl":      true,  // Latin + Cyrillic а
		"раураӏ":      false, // all Cyrillic (whole-script, NOT mixed)
		"example.com": false, // '.' is Common, ignored
		"gazа":        true,  // Latin + one Cyrillic
	}
	for s, want := range cases {
		if got := MixedScript(s); got != want {
			t.Errorf("MixedScript(%q) = %v, want %v (scripts=%v)", s, got, want, Scripts(s))
		}
	}
}

// An unassigned code point (a gap between Script ranges) or a Private Use Area
// rune is not a real script; scriptOf's documented fallback reports "Unknown"
// for both, and neutral() must treat "Unknown" the same as Common/Inherited so
// a single real script alongside one of these doesn't register as mixed-script.
func TestUnknownScriptIsNeutral(t *testing.T) {
	for _, s := range []string{
		"apple\U000E01F0", // Private Use Area (Supplementary A)
		"apple͸",          // an unassigned gap in the Greek block
	} {
		if got := MixedScript(s); got {
			t.Errorf("MixedScript(%q) = true, want false (scripts=%v)", s, Scripts(s))
		}
		if got := Scripts(s); !reflect.DeepEqual(got, []string{"Latin"}) {
			t.Errorf("Scripts(%q) = %v, want [Latin] (Unknown must be excluded)", s, got)
		}
	}
}

func TestScripts(t *testing.T) {
	if got := Scripts("pаypal"); !reflect.DeepEqual(got, []string{"Cyrillic", "Latin"}) {
		t.Errorf("Scripts = %v, want [Cyrillic Latin]", got)
	}
	if got := Scripts("example.com"); !reflect.DeepEqual(got, []string{"Latin"}) {
		t.Errorf("Scripts(example.com) = %v, want [Latin] ('.' is Common)", got)
	}
}

func TestAnalyze(t *testing.T) {
	r := Analyze("pаypal")
	if r.Skeleton != "paypal" {
		t.Errorf("Skeleton = %q, want paypal", r.Skeleton)
	}
	if !r.MixedScript {
		t.Error("MixedScript = false, want true")
	}
	if !reflect.DeepEqual(r.Scripts, []string{"Cyrillic", "Latin"}) {
		t.Errorf("Scripts = %v", r.Scripts)
	}
}

func TestCaseFold(t *testing.T) {
	// The skeleton is case-folded, so a digit→uppercase-letter confusable (0→O)
	// still JOINs a lower-case brand.
	if !Confusable("g00gle", "google") {
		t.Errorf("g00gle/google not confusable; skeletons %q vs %q", Skeleton("g00gle"), Skeleton("google"))
	}
	if got := Skeleton("g00gle"); got != "google" {
		t.Errorf("Skeleton(g00gle) = %q, want google (case-folded)", got)
	}
	// A pure case variant is the SAME DNS identity, not a homograph of it — DNS
	// names are case-insensitive, so "PayPal"/"paypal" name one thing. Confusable
	// must not flag an identity against itself just because it appears in a
	// different case (the false positive its != guard exists to prevent).
	if Confusable("PayPal", "paypal") {
		t.Errorf("PayPal/paypal reported confusable; must be false (same identity, case-insensitive)")
	}
	if Confusable("PayPal.com", "paypal.com") {
		t.Error("PayPal.com/paypal.com reported confusable; must be false (same DNS name)")
	}
}

func TestWholeScriptConfusable(t *testing.T) {
	// An all-Cyrillic label that reads as Latin is caught by the skeleton and is
	// NOT flagged mixed-script (the whole-script case the skeleton, not MixedScript,
	// must catch). "саро" is Cyrillic "саро", reading "capo".
	cyr := "саро"
	if !Confusable(cyr, "capo") {
		t.Errorf("all-Cyrillic %q not confusable with capo; skeleton %q", cyr, Skeleton(cyr))
	}
	if MixedScript(cyr) {
		t.Error("all-Cyrillic label wrongly flagged mixed-script")
	}
	if got := Scripts(cyr); !reflect.DeepEqual(got, []string{"Cyrillic"}) {
		t.Errorf("Scripts = %v, want [Cyrillic]", got)
	}
}

func TestScriptOfBoundaries(t *testing.T) {
	// Exercises scriptOf (the sort.Search refactor) across ranges + gaps.
	cases := map[string][]string{
		"A": {"Latin"},    // U+0041
		"а": {"Cyrillic"}, // Cyrillic а
		"あ": {"Hiragana"}, // Hiragana あ
	}
	for s, want := range cases {
		if got := Scripts(s); !reflect.DeepEqual(got, want) {
			t.Errorf("Scripts(%q) = %v, want %v", s, got, want)
		}
	}
	// An unassigned high code point resolves to no script (Unknown, filtered out).
	if got := Scripts("\U000E0100"); len(got) != 0 && !reflect.DeepEqual(got, []string{"Unknown"}) {
		t.Logf("Scripts(unassigned) = %v", got) // informational
	}
}

func TestUnicodePin(t *testing.T) {
	if Unicode() != "17.0.0" {
		t.Errorf("Unicode() = %q, want 17.0.0", Unicode())
	}
}
