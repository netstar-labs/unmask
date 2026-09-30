# Audit findings — netstar-labs/unmask

**Pass**: A1 adversarial-audit (four report-only auditors + adversarial verify
pass) plus a least-code pass, run 2026-09-29. Scope: owned code (`doc.go`,
`unmask.go`, `internal/gen/main.go`, `app/unmask/main.go`) — `tables.go`
(8871 lines, "Code generated ... DO NOT EDIT") is frozen, out of scope, same
treatment as idna's vendored tree and sanitize's PSL data.

**Baseline** (`go vet`, `staticcheck@2025.1.1` per the pinned CI, `gofmt -l`,
`deadcode -test ./...`): fully clean — no findings at all, unlike the two
prior repos in this session. (Note: the exact pinned `staticcheck@2025.1.1`
errors out against the current Go toolchain's export-data format on this
machine — a known, pre-existing local tool/toolchain mismatch, unrelated to
this repo; verified clean instead with a locally-built current `staticcheck`.)

## Top line

One real security finding (a false-positive gap in the package's central
anti-false-positive guarantee), one MINOR correctness gap, one MINOR
robustness gap in the table generator, and a cross-validated dedup finding —
all fixed on this branch. Every fix carries a regression test, sabotage-
verified. The security finding was independently adversarially verified
*before* being fixed, specifically because it reversed an existing, passing
test's stated expectation.

## CONFIRMED and fixed

| # | Severity | Finding | Commit |
|---|---|---|---|
| 1 | **high** | `Confusable`'s guard was byte-exact (`a != b`), not case-fold-insensitive, so two case variants of the SAME DNS identity (`"PayPal.com"` / `"paypal.com"`) were reported as confusable with each other — the exact false-positive class the guard exists to prevent. Reversed an existing test's stated expectation; independently adversarially verified before fixing. | `4c6129e` |
| 2 | low | `Unknown` script (unassigned code points, gaps between ranges) wasn't treated as neutral, so a label with one real script plus an unassigned/PUA rune registered as mixed-script. | `13ea073` |
| 3 | low | `internal/gen/main.go`'s hex parser silently returned U+0000 on failure instead of erroring, and the generator never validated its own "sorted, non-overlapping" output invariant before writing `tables.go`. Not reachable against current data; a latent robustness gap. | `ee82b31` |

Full reproductions and the adversarial-skeptic verification transcript for
finding 1: `audit-correctness.md`.

## Applied (low-risk, no public-behavior change)

- **Dedup** (auditors B + A + least-code, triple cross-validated): the CLI's
  args-vs-stdin dispatch, duplicated between `skeleton()` and `check()`, plus
  its nested reader-scan idiom repeated a third time in `readLines`.
  `audit-dedup.md`.
- **Docs**: `Map()` was undocumented everywhere outside its own doc comment
  (same pattern as idna's `IDNAVersion` from a prior audit pass on a sibling
  repo) — documented in 4 places. Every description of `Confusable`'s guard
  updated to match the case-fold fix, in the same pass that changed the code.
  `audit-docs.md`.

Re-validation after every commit: `go build`, `go vet`, `staticcheck`,
`gofmt -l`, `go test -race` all green.

## Considered and NOT applied

- Auditor A's proposal to collapse `MixedScript` into
  `len(Scripts(s)) >= 2` — logically equivalent but loses an early-exit and
  zero-allocation property in a function called in the CLI's bulk-processing
  hot path; the auditor's own report flagged this as a real, if unbenchmarked,
  performance tradeoff. `audit-simplify.md`.
- `Map`'s zero-in-tree-callers status — investigated (least-code + doc-drift),
  found to be a deliberate, documented-in-the-same-commit public accessor, not
  dead code. Kept, now documented. `audit-simplify.md` / `audit-docs.md`.

## Dimension reports

- `audit-simplify.md` — auditor A + least-code
- `audit-dedup.md` — auditor B
- `audit-correctness.md` — auditor C, with the adversarial-skeptic verification
- `audit-docs.md` — auditor D
