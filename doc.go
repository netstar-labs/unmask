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
// false positive. Detection is a JOIN with a guard, which [Confusable] encodes:
// Confusable(candidate, brand) == (skeleton(candidate) == skeleton(brand) &&
// !EqualFold(candidate, brand)), evaluated against the target list that is the
// actual product. The guard is case-fold-, not byte-, exact — DNS names are
// case-insensitive, so "PayPal.com" and "paypal.com" are one identity, not two;
// a byte-exact guard would still flag the brand against itself the moment it
// appears in a different case anywhere in the pipeline.
//
// [MixedScript] is the second, independent signal: a label mixing scripts (Latin +
// Cyrillic) is suspicious on its own, with no target list. A whole-script
// confusable — a label rendered entirely in one script that reads as another — is
// NOT mixed-script and is caught only by the skeleton; ship both. Note MixedScript
// is a RAW signal: it fires on any 2+ scripts, including legitimate multilingual
// labels (Han + Hiragana Japanese, say) — UTS-39's restriction-level status, which
// permits those combinations, is deferred (see Scope), so weight MixedScript with a
// target list rather than treating it as a verdict on its own.
//
// # Input contract
//
// unmask operates on the U-label — the pre-punycode Unicode host, as normie's
// Display field or idna.ToUnicode yields — normalised to NFC. idna.ToUnicode
// already applies NFC, so the recommended pipeline satisfies this; a caller that
// skeletonises a raw host must NFC-normalise it first, or a decomposed form
// (base + combining mark) will not collide with its precomposed look-alike ([Skeleton]
// folds confusables, not NFD — see Scope). Case need NOT be normalised: the
// skeleton is lower-cased.
//
// It is pure Go with zero external dependencies; the confusable and script tables
// are generated from the Unicode data files (see internal/gen). Because the
// skeleton is a clustering key and never a stored key, those tables update freely
// with each Unicode release — the opposite discipline from idna's pinned UTS-46
// mapping.
//
// # Scope (v1)
//
// The skeleton applies the confusables.txt MA mapping and lower-cases. Full UTS-39
// additionally folds NFD around the mapping (canonical-equivalence of combining
// marks) and defines a five-level restriction status from the augmented script sets
// (which would stop [MixedScript] over-firing on legitimate multilingual labels);
// both are deferred — the MA mapping covers the dominant domain-label homographs,
// and NFC input + [MixedScript] cover the rest. See the README's "Scope (v1)".
package unmask
