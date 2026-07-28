// Package unmask detects homograph / confusable look-alikes by the UTS-39
// skeleton: reduce a label to a canonical form under which every visual
// look-alike collides (`pаypаl` with Cyrillic а skeletonises to `paypal`), then
// JOIN that skeleton against a brand/target list to find squats. It is the
// UTS-39 (confusables) half of the netstar host toolkit; idna is the UTS-46
// (A-label / lookup key) half. The two are deliberately separate — see below.
//
// The load-bearing rule:
//
//	The skeleton is MANY-TO-ONE and is NEVER a lookup key or hash input.
//
// `раypal.com` (Cyrillic) skeletonises to the same string as the real
// `paypal.com`. Using skeleton==skeleton as a block rule would flag the
// legitimate brand because someone registered its homograph — a catastrophic
// false positive. Detection is a JOIN with a not-equal guard, which [Confusable]
// encodes: Confusable(candidate, brand) == (skeleton(candidate) == skeleton(brand)
// && candidate != brand), evaluated against the target list that is the actual
// product.
//
// [MixedScript] is the second, independent signal: a label mixing scripts (Latin +
// Cyrillic) is suspicious on its own, with no target list. A whole-script
// confusable — a label rendered entirely in one script that reads as another — is
// NOT mixed-script and is caught only by the skeleton; ship both.
//
// unmask operates on the U-label (the pre-punycode Unicode host — from normie's
// Display field or idna.ToUnicode), and the caller lower-cases first. It is pure
// Go with zero external dependencies; the confusable and script tables are
// generated from the Unicode data files (see internal/gen). Because the skeleton
// is a clustering key and never a stored key, those tables update freely with each
// Unicode release — the opposite discipline from idna's pinned UTS-46 mapping.
//
// # Scope (v1)
//
// The skeleton applies the confusables.txt MA mapping. Full UTS-39 additionally
// folds NFD around the mapping (canonical-equivalence of combining marks) and
// defines a five-level restriction status from the augmented script sets; both are
// deferred — the MA mapping alone covers the dominant domain-label homographs, and
// [MixedScript] delivers the primary script signal. See the roadmap.
package unmask
