# TW-6901: upstream security dependency locks

These files overlay `go.mod` and `go.sum` from Argo Rollouts **v1.10.0**, commit
`d90700ae8d71d141561f0c546e19f999bb335cbd`. The Dockerfile fetches and verifies
that exact commit. No upstream application code or Kubernetes dependencies
are intentionally changed.

The only manually selected module changes are:

| Module | Upstream | Patched | Critical/high findings addressed |
| --- | --- | --- | --- |
| `google.golang.org/grpc` | 1.80.0 | 1.83.2 | CVE-2026-84445, CVE-2026-84304, GHSA-hrxh-6v49-42gf |
| `golang.org/x/crypto` | 0.53.0 | 0.56.0 | CVE-2026-56854, CVE-2026-56855, CVE-2026-78662 |

Running `go mod tidy` with Go 1.26.8 produces the accompanying checksums and
required transitive updates, including OpenTelemetry **1.44.0**, which fixes
CVE-2026-29181 (fixed since 1.41.0). It also pulls newer Google SDK and `x/*`
modules; these are dependency requirements, not a general dependency refresh.
The separate metric plugin uses the repository-root module files.

To refresh these locks, use a disposable checkout of the exact upstream commit,
apply the two version changes above, and run `go mod tidy` and `go mod verify`.
Review the diff against the original release before replacing these files.
Do not run `go get -u` or use floating module versions in the Dockerfile.

The dashboard's UI is rebuilt using the upstream frozen pnpm lockfile and
pinned Node image; Node and its build dependencies are not in the final image.
Both executables are built through upstream Make targets with read-only module
resolution and a fixed build timestamp. They report `v1.10.0+d90700a.dirty`
to distinguish this dependency-patched build from the official release.

Run `make test`, `make test-upstream`, both image targets, `make scan-images`
and `make smoke-images` before publication. Allow enough Docker disk space for
both Go build architectures and the upstream unit-test compilation. The release
gate is **zero critical/high image findings**;
it is not a claim of zero vulnerabilities or a substitute for staging tests.
