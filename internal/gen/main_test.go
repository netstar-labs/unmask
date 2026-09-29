package main

import "testing"

func TestHexRune(t *testing.T) {
	if r, ok := hexRune("41"); !ok || r != 'A' {
		t.Errorf("hexRune(41) = (%q, %v), want ('A', true)", r, ok)
	}
	if _, ok := hexRune("not-hex"); ok {
		t.Error("hexRune(not-hex) = ok=true, want false")
	}
	if _, ok := hexRune(""); ok {
		t.Error("hexRune(\"\") = ok=true, want false")
	}
}

// checkNonOverlapping is the safety net that turns a malformed source row into
// a build-time failure instead of a silently-corrupted tables.go: scriptOf's
// binary search depends on ranges being sorted-by-lo and non-overlapping.
func TestCheckNonOverlapping(t *testing.T) {
	if err := checkNonOverlapping([]scriptRange{
		{0x0041, 0x005A, "Latin"},
		{0x0061, 0x007A, "Latin"},
	}); err != nil {
		t.Errorf("well-formed ranges rejected: %v", err)
	}
	if err := checkNonOverlapping([]scriptRange{
		{0x0041, 0x005A, "Latin"},
		{0x0050, 0x007A, "Cyrillic"}, // overlaps the first range
	}); err == nil {
		t.Error("overlapping ranges accepted, want an error")
	}
	if err := checkNonOverlapping([]scriptRange{
		{0x005A, 0x0041, "Latin"}, // lo > hi
	}); err == nil {
		t.Error("lo > hi range accepted, want an error")
	}
}
