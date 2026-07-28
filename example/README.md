# unmask examples

| Example | What it shows | Run |
|---|---|---|
| [confusable](confusable/main.go) | JOINing observed labels against a brand set via the UTS-39 skeleton, plus mixed-script flagging | `go run ./example/confusable` |

The skeleton is a clustering index, never a lookup key — detection is
`Confusable(candidate, brand)`, a JOIN against your target list, not
`skeleton == skeleton`. See [../doc.go](../doc.go).
