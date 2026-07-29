# unmask — user guide

## Library

```go
import "github.com/netstar-labs/unmask"

unmask.Skeleton("pаypаl")            // "paypal"   — Cyrillic а folds to Latin a
unmask.Skeleton("g00gle")            // "google"   — 0 → O, then lower-cased
unmask.Skeleton("amazon")            // "arnazon"  — m → rn (a multi-rune prototype)
unmask.Confusable("pаypаl", "paypal") // true      — same skeleton, distinct strings
unmask.Confusable("paypal", "paypal") // false     — identical; a brand is not its own squat
unmask.Scripts("pаypаl")             // ["Cyrillic" "Latin"]
unmask.MixedScript("pаypаl")         // true       — Latin + Cyrillic
unmask.Analyze("pаypal")             // {Skeleton:"paypal", Scripts:["Cyrillic" "Latin"], MixedScript:true}
unmask.Unicode()                     // "17.0.0"   — the tables' Unicode release
```

### The calls

- **`Skeleton(s) string`** — the UTS-39 confusable skeleton: each rune replaced by
  its confusables.txt MA prototype, the whole result lower-cased. A **clustering
  index, never a lookup key** (see below). The prototype can be multi-rune (`m`→`rn`)
  and can invert case before folding (`0`→`O`→`o`), so do not assume length or that
  digits survive (`1`→`l`).
- **`Confusable(a, b) bool`** — the detection test: `a != b && Skeleton(a) ==
  Skeleton(b)`. Use it as a JOIN, `Confusable(candidate, brand)`. The `a != b` guard
  is what keeps it from flagging the legitimate brand against its own homograph.
- **`Scripts(s) []string`** — the distinct Unicode scripts in `s`, excluding the
  neutral `Common`/`Inherited` (digits, punctuation, combining marks), sorted. A
  single entry means single-script.
- **`MixedScript(s) bool`** — whether `s` mixes 2+ non-neutral scripts. An
  independent, target-list-free homograph signal — but a *raw* one (see the contract).
- **`Analyze(s) Report`** — `{Skeleton, Scripts, MixedScript}` in one pass, for when
  you want all three signals.
- **`Unicode() string`** — the Unicode release the tables were generated from.
  Informational (the skeleton is never a stored key), so it bumps freely — not a
  migration stamp.

### The load-bearing rule: JOIN, never look up

The skeleton is **many-to-one** — that collapsing is its purpose — so it is **never a
lookup key or a block rule**. `раypal.com` (Cyrillic) skeletonises to the same string
as the real `paypal.com`; a rule that blocked "skeleton == `paypal`" would block the
legitimate brand the moment a squatter registered its look-alike. Detection is a JOIN
with a not-equal guard against your target list:

```go
func isSquatOf(candidate string, brands []string) (string, bool) {
    for _, b := range brands {
        if unmask.Confusable(candidate, b) {
            return b, true
        }
    }
    return "", false
}
```

For more than a handful of brands, bucket them by `Skeleton` **once** and make the
JOIN an O(1) map lookup on the candidate's skeleton — the recommended pattern:

```go
// Build once. Multiple brands can share a skeleton bucket (that is expected).
index := map[string][]string{}
for _, b := range brands {
    sk := unmask.Skeleton(b)
    index[sk] = append(index[sk], b)
}

// Query: JOIN, keeping the not-equal guard so a brand never matches itself.
func squatsOf(candidate string) []string {
    var hits []string
    for _, b := range index[unmask.Skeleton(candidate)] {
        if candidate != b { // the Confusable not-equal guard, inline
            hits = append(hits, b)
        }
    }
    return hits
}
```

`MixedScript` is orthogonal — evaluate it per candidate regardless of the JOIN, since
it catches suspicious labels that are not (yet) in your brand list, while the JOIN
catches whole-script confusables that mix no scripts. Ship both.

### Contract and cost

- **unmask normalises very little.** The caller supplies the **U-label** — the
  pre-punycode Unicode host, as normie's `Display` or `idna.ToUnicode` yields —
  **normalised to NFC** (`idna.ToUnicode` already does this). unmask does *not* decode
  punycode, strip to eTLD+1, or NFC-normalise. It *does* lower-case (the skeleton
  folds case), so only case need not be normalised. Skeletonising a raw, non-NFC host
  can miss a decomposed look-alike (v1 folds the MA mapping, not NFD — supply NFC).
- **`MixedScript` is a raw signal.** It fires on any 2+ non-neutral scripts, including
  legitimate multilingual labels (Han + Hiragana). Weight it with a target list; do
  not treat it as a standalone verdict. UTS-39's restriction-level status, which would
  permit those combinations, is deferred in v1.
- **Deterministic and stateless.** No configuration, no shared state, no locale
  dependence — every function is safe for concurrent calls. The tables are read-only.
- **Data-agnostic.** unmask owns no brand list. Any `[]string` is a valid target list;
  the consumer owns the target projection and the normalisation.

## CLI

Build: `go build -o unmask ./app/unmask` (standalone: `GOWORK=off`).

```sh
# skeleton + scripts + mixed-script per label (a per-label diagnostic view)
unmask skeleton pаypаl g00gle amazon paypal 12345
#   pаypаl  paypal   Cyrillic,Latin  true
#   g00gle  google   Latin           false
#   amazon  arnazon  Latin           false
#   paypal  paypal   Latin           false
#   12345   l2345    -               false     (all script-neutral → scripts "-"; digit 1 folds to l)

# labels one per line on stdin when no arguments are given
printf 'pаypаl\nаpple\n' | unmask skeleton

# JOIN labels against a brand list; report the actionable hit per label
unmask check -t brands.txt pаypаl g00gle amazon paypal-login
#   pаypаl  paypal  confusable        (skeletons match, strings differ)
#   g00gle  google  confusable
#   (amazon exact-matches a brand → not a squat, skipped; paypal-login → no hit, skipped)

# a mixed-script label absent from the brand list is still surfaced
unmask check -t brands.txt bаnk
#   bаnk    -       mixed-script      (Cyrillic а; no brand JOIN, but mixes scripts)

# -all also prints the misses, marked with -
unmask check -t brands.txt -all pаypаl amazon paypal-login
#   pаypаl        paypal  confusable
#   amazon        -       -
#   paypal-login  -       -

unmask version
#   unmask dev (none), Unicode 17.0.0
```

`skeleton` prints one tab-separated line per label: `label <tab> skeleton <tab>
scripts <tab> mixed` (scripts joined by `,`, or `-` when the label is entirely
script-neutral).

`check` reports the actionable hit per label: a confusable brand JOIN when one exists
(`label <tab> brand <tab> confusable`), else the target-independent mixed-script
signal when the label mixes scripts (`label <tab> - <tab> mixed-script`). The
confusable row is preferred because it names the brand; a label with neither signal is
skipped unless `-all` is set.

| Command | Flags | Meaning |
|---|---|---|
| `skeleton` | — | print `label<tab>skeleton<tab>scripts<tab>mixed`; labels from args or stdin |
| `check` | `-t <file>` (required), `-all` | JOIN labels against the brand list; print `label<tab>brand<tab>kind` per hit (kind: `confusable` / `mixed-script`) |
| `version` | — | binary version + build revision + Unicode release |

## Regenerating on a new Unicode release

```sh
go run ./internal/gen -v 16.0.0   # fetch + regenerate tables.go, then commit
```

Because the skeleton is a clustering index (never a stored lookup key), the tables
update freely every Unicode release with no migration — the opposite of `idna`'s
pinned UTS-46 mapping. `Unicode()` reports the current release for provenance; treat
it as informational, not a compatibility contract.

## When to reach past unmask

unmask is the glyph axis and only that. If you need to catch a **typo** (`paypa1`,
`gooogle`) reach for the edit-distance sibling `twist`; a **homophone** (`fone`,
`kwik`) is the phonetic sibling `echo`; and the **A-label / lookup key** (punycode
round-trip, UTS-46 mapping) is `idna`. A brand monitor runs all four and combines the
terms. Within unmask itself, the deferred refinements — the NFD fold and the UTS-39
restriction-level status — are noted in
[architecture.md](architecture.md) § "Deliberately out (YAGNI)"; build them when a
corpus proves the residue matters.
