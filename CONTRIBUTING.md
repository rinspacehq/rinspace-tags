# Contributing to Rinspace Tags

Rinspace Tags accepts focused pull requests that improve the relationship
kernel, contracts, adapters, tests, documentation, or reproducible tooling.

Before opening a pull request:

1. Read [CONTRIBUTION-LICENSE.md](CONTRIBUTION-LICENSE.md) and [DCO](DCO).
2. Add a matching `Signed-off-by` trailer to every commit with `git commit -s`.
3. Run the checks relevant to your change from the repository root. The full
   local check set is listed below.
4. Use synthetic fixtures. Do not submit production data, credentials, private
   hostnames, access tokens, cookies, or internal recovery material.
5. Explain which relationship invariant changes and include a focused test.

Classification parents, aliases, lifecycle successors, authored references,
content associations, and semantic statements have different meanings. A
proposal that combines them into one untyped edge should first explain how it
preserves those meanings and their governance histories.

Merging a public contribution does not automatically deploy it to
[rinspace.com](https://rinspace.com). Rinspace adopts an exact reviewed release
through its private integration and production gates.

## CI checks

Pull requests report each validation category separately. Open the named
failed check to see its focused log; the required `go` check is only the final
summary and succeeds after every category passes.

| Pull request check | Local command |
| --- | --- |
| Module files | `go mod tidy` and `git diff --exit-code -- go.mod go.sum` |
| Core tests | `go test -p 1 -count=1 ./core/...` |
| Contract tests | `go test -p 1 -count=1 ./contracts/...` |
| CLI tests | `go test -p 1 -count=1 ./cmd/...` |
| Static analysis | `go vet -p 1 ./...` |
| Build | `go build -p 1 ./...` |
| Source boundary | Reject private workspace or private module references |
| Synthetic graph | `go run ./cmd/rinspace-tagsctl validate -file examples/synthetic-knowledge-base/graph.json` |

CI sets `GOMAXPROCS=1`, runs at most two categories at once, and does not stop
the remaining categories after the first failure. This makes independent
errors visible without creating an unbounded test burst.
