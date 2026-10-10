# Contributing to Rinspace Tags

Rinspace Tags accepts focused pull requests that improve the relationship
kernel, contracts, adapters, tests, documentation, or reproducible tooling.

Before opening a pull request:

1. Read [CONTRIBUTION-LICENSE.md](CONTRIBUTION-LICENSE.md) and [DCO](DCO).
2. Add a matching `Signed-off-by` trailer to every commit with `git commit -s`.
3. Run `go test ./...` from the repository root.
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
