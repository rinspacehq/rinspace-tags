# Rinspace Tags repository instructions

This repository is the public source authority for the vendor-neutral Tags
core only after a coordinated Rinspace source-authority cutover is recorded.
Until then, follow the current repository status in `README.md` and release
notes.

- Keep Tag identity, name context, classification parents, aliases, lifecycle
  successors, authored references, product associations and governed semantic
  statements as distinct concepts.
- Do not add a hard-coded `tag_type` or infer identity from a name or slug.
- Keep the core independent of Rinspace accounts, cookies, CloudBase,
  production Control Plane types, private hostnames and deployment secrets.
- Put product-specific identity, social behavior, content tables, operations
  and deployment policy in the consuming product adapter.
- Use synthetic fixtures. Never commit production data or credentials.
- Preserve deterministic validation and stable machine error codes.
- Run `GOMAXPROCS=1 go test -p 1 ./...`, `go vet ./...` and `go build ./...`.
- Sign commits according to `DCO` and `CONTRIBUTION-LICENSE.md`.
- A merge to `main` does not authorize a Rinspace private dependency upgrade or
  production deployment. Consumers pin reviewed immutable releases.
