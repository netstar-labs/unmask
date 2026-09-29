# unmask

Homograph / confusable detection for Go via the **UTS-39 skeleton** — reduce a
label to a canonical form under which every visual look-alike collides, then JOIN
it against your brand list. Pure Go, **zero external dependencies**; tables
generated from the Unicode data files.

```go
unmask.Confusable("pаypаl", "paypal")  // true  — Cyrillic а reads as Latin a
unmask.Confusable("g00gle", "google")  // true  — skeleton is case-folded (0 ⇄ O)
unmask.Skeleton("g00gle")              // "google"   (digit 0 ⇄ O, lower-cased)
unmask.Skeleton("amazon")              // "arnazon"  (m ⇄ rn, a multi-rune confusable)
unmask.MixedScript("pаypаl")           // true  — Latin + Cyrillic (strongest single signal)
unmask.Analyze("pаypal")               // {Skeleton:"paypal", Scripts:[Cyrillic Latin], MixedScript:true}
```

## The load-bearing rule

**The skeleton is many-to-one and NEVER a lookup key.** `раypal.com` (Cyrillic)
skeletonises to the same string as the real `paypal.com`; using `skeleton ==
skeleton` as a block rule would flag the *legitimate* brand because someone
registered its homograph. Detection is a JOIN with a case-fold-aware guard —
`Confusable(candidate, brand)` — against the target list that is the actual
product; the guard is case-insensitive because DNS names are ("PayPal.com" and
"paypal.com" are one identity, not two).

`MixedScript` is the second, independent signal (a Latin label with one Cyrillic
character is suspicious with no target list). A **whole-script** confusable — an
all-Cyrillic label that reads as Latin — is *not* mixed-script and is caught only
by the skeleton; ship both.

## Where it fits

unmask is the **UTS-39** (confusables) half of the netstar host toolkit;
[`idna`](https://github.com/netstar-labs/idna) is the **UTS-46** (A-label / lookup
key) half. Opposite disciplines by design:

| | `idna` (UTS-46) | `unmask` (UTS-39) |
|---|---|---|
| produces | the A-label **lookup key** | the **skeleton** (clustering index) |
| is a key? | yes | **never** — JOIN index only |
| tables | vendored, **pinned** | **generated**, updated freely |

Feed unmask the **U-label** (the pre-punycode Unicode host — from normie's
`Display` or `idna.ToUnicode`), **normalised to NFC** (`idna.ToUnicode` already
does this). Case need not be normalised — the skeleton is case-folded.
`MixedScript` is a **raw** signal: it fires on any 2+ scripts, including legitimate
multilingual labels (Han + Hiragana), so weight it with a target list rather than
treating it as a standalone verdict.

## Documentation

- **Start here** — [docs/introduction.md](docs/introduction.md) ·
  [docs/executive-summary.md](docs/executive-summary.md)
- **Deep dive** — [docs/architecture.md](docs/architecture.md)
- **Operations** — [docs/userguide.md](docs/userguide.md)
- **Examples** — [example/README.md](example/README.md)

## Layout

| File | Purpose |
|---|---|
| [unmask.go](unmask.go) | `Skeleton`, `Confusable`, `Scripts`, `MixedScript`, `Analyze`, `Map`, `Unicode` |
| [tables.go](tables.go) | generated confusable map + script ranges (Unicode 17.0.0) |
| [internal/gen/](internal/gen/main.go) | the generator — `go run ./internal/gen` (re-fetches + regenerates) |
| [doc.go](doc.go) | package doc — the JOIN rule and the UTS-46/UTS-39 split |
| [app/unmask/](app/unmask/main.go) | the CLI — `skeleton` · `check` · `version` |
| [docs/](docs/) | the introduction / executive-summary / architecture / userguide quartet |

## Regenerating on a new Unicode release

```sh
go run ./internal/gen -v 16.0.0   # fetch + regenerate tables.go, then commit
```

Because the skeleton is a clustering key (never a stored lookup key), the tables
update freely every Unicode release with no migration — the opposite of `idna`'s
pinned mapping.

## Scope (v1)

The skeleton applies the confusables.txt MA mapping and lower-cases. Full UTS-39
additionally folds NFD around it (combining-mark canonical equivalence; supply NFC
input as a stand-in) and defines a five-level restriction status (which would stop
`MixedScript` over-firing on legitimate multilingual labels) —
restriction status from the augmented script sets; both are deferred — the MA
mapping alone covers the dominant domain-label homographs and `MixedScript`
delivers the primary script signal.

Go module `github.com/netstar-labs/unmask`. Standard library only. Build with
`GOWORK=off`.
