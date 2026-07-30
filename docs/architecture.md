# unmask — architecture

A flat library (`package unmask`) at the repo root, plus a thin CLI under
`app/unmask` and a generator under `internal/gen`. The pipeline is: U-label →
skeleton (per-rune confusable fold + lower-case) → JOIN against a brand target list,
with a parallel script analysis (`Scripts` / `MixedScript`) as an independent signal.

## Data flow

```
label (U-label, NFC) ─▶ Skeleton: for each rune, confusable[r] or r ─▶ lower-case ─▶ skeleton
                                                                            │
                        brand target list ─▶ (bucket by Skeleton, caller-owned)
                                                                            │
                        Confusable(label, brand) = skeleton(label)==skeleton(brand) && label!=brand
                                                                            ▼
                                                       JOIN hit (label is a squat of brand)

label ─▶ Scripts: scriptOf(r) per rune, drop Common/Inherited, dedup+sort ─▶ []script
                                                                            │
                        MixedScript = 2+ distinct non-neutral scripts ──────┘  (independent signal, no target list)
```

`Analyze` runs all three (`Skeleton`, `Scripts`, `MixedScript`) over the label in one
call and returns a `Report`.

## The skeleton (`unmask.go`)

`Skeleton(s)` is the UTS-39 confusable skeleton. It walks the label rune by rune; for
each rune it looks up the generated `confusable` map and appends the prototype string
if present, or the rune itself if not, into a pre-grown `strings.Builder`. Two
decisions carry the correctness:

- **The prototype is a string, not a rune — the mapping is one-to-many.** UTS-39's MA
  data maps some confusables to a *sequence*: `m` → `rn`, so `amazon` skeletonises to
  `arnazon` (an `m` reads as an `r` next to an `n`). The map is therefore
  `map[rune]string`, and the builder appends a string per rune. A caller cannot
  assume `len(skeleton) == len(label)`.
- **The result is lower-cased, and that is load-bearing.** Several MA prototypes are
  upper-case — the digit `0` maps to `O`, `1` can map to `l`, `I` to `l` — so without
  a final fold `g00gle` would skeletonise to `gOOgle` and fail to JOIN a lower-case
  brand `google`. Lower-casing the whole result makes the index case-insensitive so
  the JOIN is symmetric regardless of the brand's case. (Consequence worth knowing:
  `12345` skeletonises to `l2345`, because the digit `1` folds to `l` — the skeleton
  is a look-alike index, not a semantic one.)

`Confusable(a, b)` is the detection test and the entire safety story:

```go
func Confusable(a, b string) bool { return a != b && Skeleton(a) == Skeleton(b) }
```

The `a != b` guard is not a micro-optimisation — it is the reason the detector is
safe to ship. The skeleton is **many-to-one**: `раypal` (Cyrillic) and the genuine
`paypal` share a skeleton by construction. A block rule keyed on the bare skeleton
would flag the legitimate brand the moment someone registered its homograph.
`Confusable` fires only when the two *distinct* strings collide, evaluated as a JOIN
`Confusable(candidate, brand)` against the target list — never `skeleton == skeleton`
alone, and never the skeleton as a stored key.

## The generated tables + the Unicode pin (`tables.go`, `internal/gen`)

`tables.go` is generated, never hand-edited (`// Code generated … DO NOT EDIT`). It
holds three things:

- `confusable map[rune]string` — the confusables.txt MA mapping (rune → prototype
  string).
- `scriptRanges []scriptRange` — the Unicode Script property as sorted,
  non-overlapping `{lo, hi, script}` ranges from Scripts.txt.
- `unicodeVersion` — the release string these were built from (currently `17.0.0`),
  surfaced by `Unicode()`.

`internal/gen` fetches `confusables.txt` (security) and `Scripts.txt` (UCD) from
unicode.org for a pinned version, parses the MA column and the script ranges, and
emits `tables.go` via `go/format`. Crucially, **this is safe to re-run and commit on
every Unicode release** — the exact opposite of `idna`'s pinned UTS-46 vendor. A
UTS-46 A-label mapping is a *stored lookup key*: change it and old keys stop
resolving, so it must be pinned and migrated. The skeleton is a *clustering index*
computed fresh each time and never stored, so newer confusables data only ever
improves recall with no migration. `Unicode()` is therefore informational — a
provenance stamp, not a compatibility contract.

## Scripts and MixedScript (`unmask.go`)

`scriptOf(r)` resolves a rune's Unicode Script property by binary search
(`sort.Search`) over `scriptRanges`. The ranges are sorted by `lo` and
non-overlapping (hence also sorted by `hi`), so the search finds the first range with
`hi >= r` and confirms `lo <= r`; a rune in a gap or beyond the table reports
`"Unknown"`. That is O(log n) per rune with no allocation.

`Scripts(s)` collects the distinct scripts across the label, **excluding the
script-neutral `Common` and `Inherited`** (digits, punctuation, combining marks
carry no script identity), and returns them sorted. `MixedScript(s)` short-circuits:
it records the first non-neutral script it sees and returns `true` at the first
differing one, so it is O(runes) and usually stops early.

The two signals are complementary, and the boundary matters:

- **MixedScript is a raw signal.** It fires on any 2+ non-neutral scripts, including
  legitimate multilingual labels (Han + Hiragana in Japanese). UTS-39's restriction
  status — which permits those combinations — is deferred (see below), so a consumer
  weights `MixedScript` alongside a target-list JOIN rather than treating it as a
  standalone verdict.
- **A whole-script confusable is invisible to MixedScript.** A label rendered
  *entirely* in Cyrillic that reads as Latin mixes no scripts — `MixedScript` returns
  `false` — and is caught only by the `Skeleton` JOIN. Conversely a Latin label with
  a single Cyrillic look-alike is mixed-script even before any target list. Neither
  signal subsumes the other; a detector runs both.

## Complexity

Let *n* be the label length in runes (small for host/brand labels).

- **`Skeleton`** is O(*n*): one map lookup per rune into a pre-grown builder; output
  bounded by *n* × the longest prototype (a small constant).
- **`Confusable`** is two `Skeleton` calls plus a string compare — O(*n*).
- **`scriptOf`** is O(log R) over the *R* script ranges (a fixed table); `Scripts` is
  O(*n* log R) and `MixedScript` the same with early exit.
- **`Analyze`** is the sum — still O(*n* log R), one pass' worth of work per field.

For the intended workload — short, caller-normalised labels — every call is
sub-microsecond; see `bench_test.go`. A consumer that JOINs a candidate against *B*
brands does *B* `Confusable` calls unless it buckets the brands by `Skeleton` once
(the recommended pattern — see the user guide — which makes the JOIN an O(1) map
lookup on the candidate's skeleton).

## Deliberately out (YAGNI)

The v1 scope is the smallest real capability; the tempting extensions are recorded as
later refinements, not pre-built:

- **No NFD fold.** Full UTS-39 folds NFD around the MA mapping so a decomposed
  base+combining-mark form collides with its precomposed look-alike. v1 applies the MA
  mapping and lower-casing only, and pushes the canonical-equivalence step onto the
  input contract: the caller supplies NFC (which `idna.ToUnicode` already yields), and
  the MA mapping covers the dominant domain-label homographs. Build the NFD fold when
  a corpus proves the residue matters.
- **No restriction-level status.** UTS-39 defines a five-level restriction status from
  augmented script sets that would let a detector *permit* legitimate multilingual
  labels instead of raw-flagging them. It is deferred; `MixedScript` is the primary
  script signal and is weighted with a target list rather than treated as a verdict.
  (`Report` accordingly carries `Skeleton`, `Scripts`, and `MixedScript`, not a
  restriction level.)
- **No punycode / normalisation / lower-casing of input.** unmask is not `idna`. It
  does not decode `xn--…`, strip to eTLD+1, or NFC-normalise — the caller does that
  upstream and feeds the U-label. Case *is* handled (the skeleton lower-cases), so
  only case need not be normalised.
- **No built-in target-list / `Set` type.** The skeleton is the indexing primitive; a
  consumer buckets its brands by `Skeleton` in three lines and owns its target
  projection and normalisation — exactly as `snare` and `echo` consumers do. The
  rune-level accessor `Map(rune) ([]rune, bool)` exposes the raw confusable mapping for
  callers that need it (a confusability-weighted cost, a look-alike generator); it is
  additive and does not change the skeleton contract.

## Consumer wiring (future work)

unmask imports none of its consumers; they wire it in a few lines, each owning its own
target projection and query normalisation:

| Consumer | Signal | Wiring |
|---|---|---|
| Brand monitor | a `homograph_suspect` term | normalise to the U-label (idna.ToUnicode, NFC), bucket brands by `Skeleton`, then look up `Skeleton(candidate)` and weight `MixedScript`. Adding a scored term to a trained model triggers a retrain + model-version bump on the consumer side, not unmask. |
| CT-log tailer (`vigil`) | glyph filter on observed SANs | skeletonise each observed name and JOIN against the brand skeleton bucket, alongside the edit (`snare`) and phonetic (`echo`) terms. |
| `twister` homograph generator | generate → detect round-trip | every homograph variant `twister` emits for a brand must be `Confusable` with that brand — the differential test the two share. |

## Layout

| File | Purpose |
|---|---|
| [unmask.go](../unmask.go) | `Skeleton`, `Confusable`, `Scripts`, `MixedScript`, `Analyze`, `Unicode`, and the `scriptOf` / `neutral` helpers |
| [tables.go](../tables.go) | generated `confusable` map + `scriptRanges` + `unicodeVersion` (Unicode 17.0.0) — never hand-edited |
| [internal/gen/](../internal/gen/main.go) | the generator — fetches confusables.txt + Scripts.txt and regenerates `tables.go` |
| [doc.go](../doc.go) | package doc — the JOIN rule and the UTS-46 / UTS-39 split |
| [app/unmask/](../app/unmask/main.go) | the CLI — `skeleton` · `check` · `version` |
