# Audit — simpler pathways (auditor A) + least-code

Scope: owned code — `doc.go`, `unmask.go`, `internal/gen/main.go`,
`app/unmask/main.go`. `tables.go` is generated, frozen, out of scope.

## Result: already close to minimal; no simplify findings applied

Both the least-code pass and auditor A independently concluded this codebase
is already near the floor: `go build`/`vet`/`test` clean, `deadcode -test ./...`
finds zero unreachable functions, `go.mod` carries no `require` lines (the
zero-external-dependency claim is verified, not just asserted), and no
unnecessary abstraction, framework reach, or premature generalization was
found in the owned code.

The one real cross-cutting finding both dimensions surfaced — the
args-vs-stdin CLI dispatch duplication — is a dedup finding and is recorded in
`audit-dedup.md` (applied, along with the least-code pass's independent
confirmation of the same cluster).

## Considered and NOT applied

**`MixedScript(s) bool { return len(Scripts(s)) >= 2 }`** — auditor A proposed
collapsing `MixedScript`'s hand-rolled loop into a one-line delegation to
`Scripts()`, since the two are logically equivalent (verified: output-identical
for every input, including the empty string and all-neutral labels). **Not
applied**, per the auditor's own flagged tradeoff: the current `MixedScript`
short-circuits on the second distinct script with zero allocation, while
`Scripts()` always scans the full string and builds a map + slice. Delegating
would mean every standalone `MixedScript` call (including the CLI's bulk
`check` path, reading one label per stdin line, potentially many) always pays
the full-scan-plus-allocation cost instead of early-exiting, and `Analyze`
(which already calls both) would end up computing scripts twice per label
instead of once. This is a real, if unbenchmarked, performance regression in a
function's own hot path — the auditor flagged this as "risk: medium, not low"
for exactly this reason, and it changes behavior (performance characteristics)
even though it doesn't change output, which is the class of finding the audit
process leaves for the maintainer to decide rather than applying freely.

## Investigated, not removed: `Map` (least-code + doc-drift finding)

The least-code pass flagged `Map(r rune) ([]rune, bool)` as having zero callers
anywhere in this repo, its examples, or any sibling repo on disk — the
"wrapper that only forwards, nobody calls it" pattern; `deadcode` (without
`-test`) confirms it. Auditor D investigated further (see `audit-docs.md`):
`Map` was added in the same commit that wrote the full doc set, and the
README/docs already describe an intended future use ("a confusability-weighted
substitution cost, a look-alike generator"). Same pattern as the sibling `idna`
repo's `IDNAVersion` finding from a prior audit pass. **Kept, and now
documented** (it previously wasn't, anywhere outside its own doc comment) —
see the docs commit on this branch.

## Two minor CLI observations, noted, not acted on

- The CLI accepts four spellings of `version` (`version`, `-version`,
  `--version`, `-v`) with no test or caller requiring all four — harmless
  convenience-alias sprawl, effectively free.
- `TestScriptOfBoundaries`'s soft check on an unassigned code point
  (`t.Logf`, not a hard assertion) becomes moot after the Unknown-neutral fix
  (`audit-correctness.md` finding 2) — `Scripts()` now returns `[]` for that
  input, satisfying the soft check's condition without ever logging. Not
  changed, since the soft check itself is harmless and still passes.
