# Meet unmask — the look-alike, not the typo

Some phishing domains do not misspell the brand at all — they spell it in a
different alphabet. `pаypаl.com` where the `а` is Cyrillic U+0430, not Latin
U+0061; `аpple.com` with one swapped letter your eye cannot see; `g00gle` where a
zero stands in for an O. Rendered in a browser these are pixel-for-pixel the
target, yet as byte strings they are miles apart — a different code point in every
confusable position. An edit-distance metric scores them as far away (every
substituted glyph is one more edit), and a plain string compare never fires. The
homograph is a squat class with its own door, and unmask is the key.

## What it actually is

unmask is a pure-Go, zero-dependency implementation of the **UTS-39 confusable
skeleton**. It reduces a label to a canonical form — the *skeleton* — under which
every visual look-alike collapses onto one string: each rune is replaced by its
confusable prototype, so Cyrillic `а`, Greek `α`, and Latin `a` all become the same
`a`, and `pаypаl` skeletonises to exactly `paypal`. Ask "is this observed label a
homograph of a brand I protect?" and the answer becomes: *does its skeleton equal
the brand's skeleton, while the raw labels differ?* That is what `Confusable(a, b)`
tests, and it is the whole mechanism.

`Skeleton(s)` returns the canonical form, `Confusable(a, b)` is the detection test,
`Scripts(s)` lists the distinct Unicode scripts in a label, `MixedScript(s)` reports
whether it mixes two or more, and `Analyze(s)` bundles all three into one `Report`.

## The one rule that makes it safe

The skeleton is **many-to-one**: many different strings share it, by design — that
collapsing is the whole point. Which means the skeleton is **never a lookup key**.
`раypal.com` (Cyrillic) skeletonises to the very same string as the real
`paypal.com`, so a rule that blocked "any label whose skeleton is `paypal`" would
block the legitimate brand the instant a squatter registered its look-alike — a
catastrophic false positive. Detection is a **JOIN with a not-equal guard**, not a
block rule: `Confusable(candidate, brand)` fires only when the skeletons match *and*
the raw strings differ, evaluated against the brand target list that is the actual
product. Reduce, then JOIN — never reduce and look up.

## The second signal, and why you ship both

`MixedScript` is an independent detector that needs no target list: a label mixing
Latin and Cyrillic is suspicious on its own — legitimate hosts rarely span alphabets
mid-label. But it is a *raw* signal (it fires on any 2+ scripts, including a genuine
Han + Hiragana Japanese label), so weight it, don't verdict on it. And it does not
subsume the skeleton: a **whole-script** confusable — a label rendered *entirely* in
Cyrillic that reads as Latin — mixes no scripts at all, so `MixedScript` stays quiet
and only the skeleton JOIN catches it. The two signals cover different attacks; ship
both.

## The scope it keeps

unmask answers one question — *is this a glyph look-alike of a target, and does it
mix scripts?* — and refuses the neighbouring ones. It does not decode punycode,
lower-case aside, or normalise: the caller feeds it the U-label (the pre-punycode
Unicode host, as `idna.ToUnicode` yields) already in NFC. It is not an edit-distance
metric (that is the sibling `snare`) and not a phonetic one (`echo`); it is the glyph
axis, and only that. That discipline keeps it a pure function you can drop into a
detector in three lines and unit-test against Unicode's own confusables data in
isolation.

*Read next:* [executive-summary.md](executive-summary.md) ·
[architecture.md](architecture.md) · [userguide.md](userguide.md) ·
[../example/README.md](../example/README.md)
