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

func TestUnicodePin(t *testing.T) {
	if Unicode() != "15.1.0" {
		t.Errorf("Unicode() = %q, want 15.1.0", Unicode())
	}
}
