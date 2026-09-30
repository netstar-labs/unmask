# Audit — duplication / dedup (auditor B)

Scope: owned code — `doc.go`, `unmask.go`, `internal/gen/main.go`,
`app/unmask/main.go`.

## Fixed in this pass

**The args-vs-stdin dispatch loop, duplicated between `skeleton()` and
`check()`** (`app/unmask/main.go`), plus its nested reader-scan idiom repeated
a third time in `readLines`. Cross-validated independently by auditor A and
the least-code pass — three of five auditors converged on this exact cluster.

Extracted `forEachLabel` (args-vs-stdin dispatch, shared by both subcommands)
and `scanLines` (the underlying reader-scan idiom). `readLines`'s blank-line
filter moved from an inline check in the scan loop to its own callback — the
one place a caller of `scanLines` needs it (both subcommands' `emit` closures
already guard against empty labels themselves).

Verified behaviorally identical: traced every branch by hand, built the CLI
binary, and smoke-tested both subcommands across args and stdin input
(`skeleton`, `check -t` with and without `-all`), comparing output line-for-
line against the pre-refactor shape. No `app/unmask` test file existed before
this to pin the internal shape, so the refactor needed no test updates.

## Considered and NOT recommended

- The scanner-buffer-sizing one-liner shared verbatim between
  `newLineScanner` (`app/unmask/main.go`) and `forEachRow`
  (`internal/gen/main.go`) — the CLI and the generator are separate `main`
  packages with zero existing coupling; introducing a shared internal package
  to save one line isn't worth the new dependency.
- The blank-guard at the top of both subcommands' `emit` closures — the rest
  of the two closures is substantively different (one formats an `Analyze`
  report, the other JOINs against a brand list); extracting just the guard
  buys nothing on its own.
- Per-rune iteration in `unmask.go` (`Skeleton`, `Scripts`, `MixedScript` each
  open with `for _, r := range s`) — each loop body does fundamentally
  different work, and `Analyze` already composes all three. Merging into one
  pass would be a performance change entangling three independently-testable
  behaviors, not a duplication fix.
- `check()`'s brand loop recomputing `Skeleton(b)` per brand per label rather
  than precomputing a `skeleton→brand` map once — a real possible optimization,
  but the author already reasoned about the one behavior change a map would
  introduce (which of two skeleton-colliding brands wins), so left as a
  documented, acknowledged tradeoff, not a finding to act on.
- The generator's `scriptRange` struct declared once for its own in-memory use
  and again as source text emitted into `tables.go` — two different artifacts
  by design (one compiles the generator, the other is a separate frozen file),
  not duplication to collapse.
