# unmask — executive summary

**What it is.** A small, dependency-free Go library that detects homograph /
confusable look-alike labels by the **UTS-39 skeleton**. Reduce a label to a
canonical form under which every visual look-alike collides (`Skeleton`), then JOIN
that skeleton against a brand target list (`Confusable`) to find squats. It is the
glyph sibling of `snare` (edit distance) and `echo` (phonetic) — the same "reduce to
a key, then match" shape, over *appearance* instead of spelling or sound.

**Why it exists.** A homograph squat — `pаypаl` with a Cyrillic `а`, `аpple`,
`g00gle` — is a name that *renders* as a brand while being a different byte string in
every confusable position. It defeats the other two axes: it is several substitutions
from its target, so a typo budget rejects it, and it is not a homophone, so a
phonetic coder is irrelevant. The only principled route to the homograph class is to
fold look-alikes onto one canonical form and compare *that*, which is the one thing
unmask does.

**What you get.**
- **`Skeleton(s) string`** — the UTS-39 confusable skeleton: each rune replaced by
  its confusables.txt MA prototype, lower-cased so the index is case-insensitive
  (`pаypаl` → `paypal`, `g00gle` → `google`, `amazon` → `arnazon` via the `m`→`rn`
  multi-rune prototype). A clustering index, never a lookup key.
- **`Confusable(a, b) bool`** — the correct detection test: the skeletons match *and*
  the two are distinct identities (case-fold-insensitive, since DNS names are
  case-insensitive — `"PayPal"`/`"paypal"` is one identity, not two). Used as a
  JOIN against the brand list; the guard is what stops it flagging the
  legitimate brand against its own homograph — or against itself in another case.
- **`Scripts(s) []string` / `MixedScript(s) bool`** — the distinct Unicode scripts in
  a label (excluding the neutral Common/Inherited/Unknown) and whether it mixes 2+ —
  an independent, target-list-free homograph signal.
- **`Analyze(s) Report`** — all three (`Skeleton`, `Scripts`, `MixedScript`) in one
  pass.
- **`Map(r) ([]rune, bool)`** — the raw per-rune confusable prototype `Skeleton` is
  built on, for callers building their own tooling directly on the mapping.
- **`Unicode() string`** — the Unicode release the tables were generated from
  (informational, not a migration stamp — the skeleton is never a stored key).
- **No dependencies, no state, no configuration** — standard library only, tables
  generated from the Unicode data files, a pure function of its input, safe for
  concurrent use.

**Where it fits.** unmask is the *algorithm* layer, coupled to nothing. It is the
**UTS-39** (confusables) half of the netstar host toolkit; `idna` is the **UTS-46**
(A-label / lookup key) half — opposite disciplines by design (`idna` produces a
*pinned* lookup key; unmask produces a *freely regenerated* clustering index).
Within the brand matcher family it is the third axis:

| Axis | Repo | Distance |
|---|---|---|
| edits | `snare` | Damerau-Levenshtein |
| sound | `echo` | Double Metaphone / Soundex |
| **glyphs** | **`unmask`** | **UTS-39 skeleton** |

A brand monitor scores a candidate on all three and combines the terms; unmask is the
glyph term. Feed it the caller-normalised U-label (from normie's `Display` or
`idna.ToUnicode`) in NFC; it owns no target list and no normalisation.

**What it is not.** Not an edit-distance metric (`snare`), not a phonetic one
(`echo`), not a punycode decoder or lookup-key mapper (`idna`), not a resolver or
classifier. And in v1, not full UTS-39: it applies the MA mapping plus lower-casing
but defers the NFD fold (supply NFC input) and the five-level restriction status
(use `MixedScript` weighted by a target list). It answers one question — is this a
glyph look-alike of a target — and only that.
