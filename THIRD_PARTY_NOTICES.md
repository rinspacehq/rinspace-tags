# Third-Party Notices

The relationship kernel currently depends on the following third-party module:

| Component | Version | License | Purpose |
| --- | --- | --- | --- |
| `golang.org/x/text` | `v0.15.0` | BSD-3-Clause | Unicode NFKC normalisation and case folding |
| `golang.org/x/mod` | `v0.8.0` | BSD-3-Clause | Module-graph dependency of `x/text`; not imported by the runtime packages |
| `golang.org/x/sys` | `v0.5.0` | BSD-3-Clause | Module-graph dependency of `x/text`; not imported by the runtime packages |
| `golang.org/x/tools` | `v0.6.0` | BSD-3-Clause | Module-graph dependency of `x/text`; not imported by the runtime packages |

These Go project modules use the same upstream BSD-3-Clause notice, preserved at
[`third_party/golang.org/x/text/LICENSE`](third_party/golang.org/x/text/LICENSE).

Dependency versions and transitive modules are fixed by `go.mod` and `go.sum`.
Release candidates will also carry a machine-readable SBOM and provenance.
