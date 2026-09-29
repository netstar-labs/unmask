# Audit — correctness + optimization (auditor C)

Scope: owned code — `doc.go`, `unmask.go` (133 lines), `internal/gen/main.go`
(184 lines, the table generator), `app/unmask/main.go` (190 lines, the CLI).
`tables.go` (8871 lines, generated, "DO NOT EDIT") is frozen, out of scope.

## CONFIRMED and fixed

### 1. [SECURITY, sev:high] `Confusable`'s guard was byte-exact, not case-fold-insensitive

`Confusable(a, b) = a != b && Skeleton(a) == Skeleton(b)`. DNS names are
case-insensitive, so `"PayPal.com"` and `"paypal.com"` are the *same* identity,
not two different ones — but the byte-exact `a != b` guard let that pair
through (both sides are pure ASCII, so they trivially share a lower-cased
skeleton), reporting `Confusable("PayPal.com", "paypal.com") == true`.

This is precisely the "catastrophic false positive" `doc.go` describes the
guard as existing to prevent — just arriving through the case channel instead
of the Unicode-homograph channel. The shipped CLI's `check()` always JOINs
exactly the two sources where this bites: a human-typed brand list against a
harvested candidate stream (e.g. from CT logs, which commonly normalize case)
that could include the brand's own real domain in a different case.

**Adversarially verified before fixing** (this finding directly reversed an
existing, passing test — `TestCaseFold`'s `"PayPal"`/`"paypal"` assertion,
labeled "case variants collapse"): an independent skeptic checked whether pure
ASCII-case variation was meant to be its own suspicious signal (no doc anywhere
frames it that way), whether the shipped CLI ever compares two same-source
strings where such a signal could apply (it doesn't — `check()` only ever
JOINs brand-list-vs-candidate, different sources), and whether fixing the
guard's direction (rather than `Skeleton`'s lower-casing, which is independently
load-bearing for the digit-0/letter-O case) was correct. Also checked for a
new false-negative: cross-script confusable pairs (Cyrillic/Latin) never fold
together under Go's simple `strings.EqualFold`, so no genuine homograph is
newly missed.

**Fix**: `!strings.EqualFold(a, b)` instead of `a != b`. `TestCaseFold` updated:
the `PayPal`/`paypal` assertion now expects `false`, with an explanatory
comment (the old comment, "case variants collapse," was describing the bug).
Sabotage-verified.

### 2. [MINOR] `Unknown` script counted as a real script

`neutral()` only excluded `Common`/`Inherited`; an unassigned code point or a
gap between Script ranges (`scriptOf`'s documented "Unknown" fallback) counted
as a distinct, real script for `Scripts()`/`MixedScript()` purposes. A label
with one real script plus a single Private-Use-Area or unassigned rune
registered as `MixedScript=true`, even though Unknown isn't a genuine script.
False-positive direction only; tempered since IDNA2008 disallows unassigned
code points in real hostnames.

**Fix**: `neutral()` also excludes `"Unknown"`. Regression test added (a PUA
rune, and a genuine unassigned gap in the Greek block — confirmed via direct
range-table introspection), sabotage-verified.

### 3. [MINOR] `internal/gen/main.go`: silent parse failures, no output-invariant check

`hexRune` returned `0` (a valid code point, U+0000) on any parse failure,
indistinguishable from a genuine successful parse to U+0000. Downstream:
- `parseConfusables`: a malformed *target* hex field silently wrote a NUL rune
  into a confusable prototype string (the *source* field's failure was
  incidentally guarded by an unrelated `src != 0` check, but the target field
  had no guard at all).
- `parseScripts`: no guard at all — a malformed hex field silently produced a
  spurious range anchored at U+0000.
- `emit`: never validated the "sorted by lo, non-overlapping" invariant
  `scriptOf`'s binary search (`unmask.go`) depends on, before writing
  `tables.go`.

Not reachable against the current, well-formed pinned Unicode 17 data — a
latent robustness gap for the next re-run against new source files, not a live
bug. But a corrupted range table would corrupt `Scripts()`/`MixedScript()`
silently at runtime with no error anywhere, in a security-detection library —
worth closing before it's needed.

**Fix**: `hexRune` now returns `(rune, ok bool)`; every call site checks `ok`
and fails the generator run via the existing `must()` fatal-error pattern.
`emit` gained `checkNonOverlapping`, verifying the invariant before writing.
No test file existed for `internal/gen` before this; added `main_test.go`
(`TestHexRune`, `TestCheckNonOverlapping` — well-formed, overlapping, and
`lo > hi` cases). Sabotage-verified.

## Clean checks performed (traced/run against the real package)

- **`scriptOf`'s binary search**: verified via direct introspection of the live
  2287-entry `scriptRanges` table — sorted by `lo`, monotonic non-decreasing
  `hi`, zero overlaps (exhaustive adjacency check). Boundary runes at several
  indices, a genuine gap (U+0378), and beyond the table's end all resolve
  correctly.
- **Multi-rune prototype expansion** (`m` → `"rn"`): no corruption when
  interleaved with a following multi-byte rune — `range`-over-`s` decodes
  source runes independently of `WriteString`'s output, no aliasing by
  construction.
- **NFC vs. NFD** (the documented Scope(v1) limitation): confirmed the failure
  mode is a false negative (fails to cluster a decomposed/precomposed pair),
  not a false positive — safe in the direction the docs promise.
- **`Confusable` symmetry**: `Confusable(a,b) == Confusable(b,a)` — guaranteed
  structurally (both operations are symmetric), no counterexample found.
- **Whole-script confusable vs. `MixedScript`**: an all-Cyrillic look-alike of
  a Latin word is correctly `MixedScript=false` while still caught by
  `Confusable` — matches the documented split of responsibilities exactly.
- **Digit/punctuation-only evasion attempt**: labels made entirely of
  Common-script characters correctly produce no scripts and `MixedScript=false`
  — not an evasion path since such labels contain no letters to substitute.
- **Fuzzing**: `FuzzSkeleton`, ~300k execs, 0 panics.
- **`app/unmask/main.go`** (pre-refactor): `readLines` closes its file via
  `defer`; both stdin paths bound line length to 1 MiB; no resource leaks or
  unvalidated-input paths found.
