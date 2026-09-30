# Audit — doc / comment vs code drift (auditor D)

Scope: README.md, doc.go, every `docs/*.md`, doc comments in `unmask.go`,
`internal/gen/main.go`, `app/unmask/main.go`.

## Fixed in this pass

**`Map` was undocumented everywhere except its own doc comment and
`docs/architecture.md`.** It's a real, working, deliberately-added function
(commit `51c6d93`, the same commit that wrote the full doc set from scratch) —
not a case of docs lagging a later addition, the author wrote the docs and
skipped this one function in the same commit. Absent from README.md's Layout
table, the README/userguide.md code examples, and executive-summary.md's
"What you get" list — every one of which enumerates the other six exported
functions individually. Same pattern as the sibling `idna` repo's
`IDNAVersion` omission from a prior audit pass. Severity: outdated/omission,
not wrong — nothing false was claimed, a real capability was just invisible
to a reader who only reads the prose docs. Fixed: one line/entry added to
each of the four locations.

**Every description of `Confusable`'s guard needed updating** once the
case-fold fix (`audit-correctness.md` finding 1) landed on this same branch —
`doc.go`, README.md, `docs/introduction.md`, and `docs/architecture.md`
(which quoted the literal old code) all explicitly said "not-equal guard" /
`a != b`. Updated all four in the same pass that changed the code, so this
audit doesn't ship new drift alongside the fixes it recorded.

## Confirmed clean (verified by execution, not just reading)

- Every README.md/`docs/userguide.md` code example reproduced exactly via a
  throwaway scratch module: `Confusable`, `Skeleton` (incl. the `12345`→`l2345`
  digit-fold edge case), `MixedScript`, `Analyze` — all match.
- Unicode version claim ("17.0.0") matches `tables.go`'s generation header,
  every doc reference, and `unmask.Unicode()`'s live return value.
- CLI usage: built with `GOWORK=off` (plain `go build` fails under the
  enclosing `go.work` — confirms the README/userguide's build instruction is
  load-bearing, not boilerplate) and ran every documented invocation
  (`skeleton` args/stdin, `check -t` with/without `-all`, `version`), diffing
  output against the docs field-for-field. All match.
- `doc.go`'s "Scope (v1)" deferred-features list (NFD folding, UTS-39
  restriction-level status): grepped the entire tree — neither exists. The
  scope doc's claim that these are genuinely absent is accurate, not stale.
- go.mod / Go version / "zero external dependencies": confirmed — no go.sum
  file exists, every import across the four owned files is standard library.
- `deadcode -test ./...`: clean. Without `-test`, only `Map` is flagged
  unreachable, which is the finding above (a deliberate, tested public
  accessor), not a bug.
- `example/README.md` and `example/confusable/main.go`: consistent with the
  root README's Examples link; builds cleanly under `GOWORK=off`.
