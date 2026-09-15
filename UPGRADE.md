# TW-6901: Rollouts v1.10.0 readiness

Tracking: https://nylas.atlassian.net/browse/TW-6901

Controller/plugin PR: https://github.com/nylas/rollouts-plugin-metric-coralogix/pull/8

The paired `unicorn-helm` candidate chart is
`chart/argo-rollouts-1.10/chart` (upstream chart 2.43.1). It requires the published
controller image digest and is not referenced by any active ApplicationSet.

## Controller and dashboard security patch, 2026-09-15

The latest candidate rebuilds **both** upstream executables from v1.10.0 commit
`d90700ae8d71d141561f0c546e19f999bb335cbd`, using the dependency locks under
`build/argo-rollouts/`. Go remains 1.26.8; gRPC is 1.83.2, `x/crypto` is 0.56.0,
and the required OpenTelemetry update is 1.44.0. No upstream application code
is changed. The dashboard UI uses the upstream frozen pnpm lockfile. A fresh,
digest-pinned distroless runtime avoids retaining the old executables in layers.

Final Linux/AMD64 image scans using
`docker scout cves --exit-code --only-severity critical,high` passed with
**zero critical and zero high findings in each image**, without suppressions.
The baseline controller and dashboard each had the seven high advisory IDs
listed below. Medium/low remediation is explicitly out of scope; this filtered
scan does not establish that medium/low findings are absent or audit bundled
JavaScript source dependencies.

Local image identities (Docker config IDs, **not published registry digests**):

| Image tag | Config ID |
| --- | --- |
| `argo-rollouts-coralogix:TW-6901-security` | `sha256:2fdd8669506d838d37d8d956f75c0fe67dfc1e0a7c7fb38c0b24d19d3fedfc9c` |
| `argo-rollouts-dashboard:TW-6901-security` | `sha256:78fd176521630eebaec3737c44ce00c96e29865ee9804d212e02fbc4c039d687` |

Uncached plugin tests pass. Both images build and report
`v1.10.0+d90700a.dirty`, marking the dependency-patched upstream source. The
local smoke checks verify AMD64, UID 999, dashboard startup, HTML, embedded
JavaScript and the HTTP-to-gRPC version endpoint using only a synthetic
kubeconfig; no cluster access is exercised.

The Docker upstream-test build ran out of Docker disk space while compiling.
The same selected upstream suites **passed uncached** on the macOS/ARM64 host
with Go 1.26.8 and the identical patched module files:

```sh
env GOTOOLCHAIN=go1.26.8 CGO_ENABLED=0 go test -mod=readonly -short -count=1 -p 2 \
  ./controller/... ./rollout/... ./metricproviders/... ./server/...
```

This covers controller/reconciliation, traffic-routing, metric-provider and
dashboard-server unit tests; it is not Kubernetes end-to-end testing. Linux
runtime validation is provided by the two AMD64 builds and image smoke checks.
No Docker data was pruned. The Docker test target now limits build parallelism.

The existing candidate Helm chart still references the upstream dashboard.
Before activation, update it to require the **patched dashboard digest** as
well as the controller digest. Neither artifact is published or deployed.
Earlier dated sections below are historical evidence, not the current scan
result or PR state. The PR is open for review, not approved for deployment.

## Security review, 2026-09-09

The initial custom image scan identified a vulnerable Go 1.26.1 standard library
in the plugin binary. At that point the plugin build targeted Go 1.26.8 and gRPC
1.83.1. That image passed local tests and container version checks. Its
Docker Scout scan reported no critical findings and seven high advisory
IDs, matching the advisory IDs found in the upstream controller scan.

An independent Docker Scout scan of the upstream linux/amd64 controller image
`quay.io/argoproj/argo-rollouts:v1.10.0` found **7 high-severity package findings**
and no critical findings. The pinned multi-platform base-image digest is
`sha256:187630ba722846bb8e079630364d3c29c8246a256be95785fd957b088f621c4e`.

| Package in upstream controller | Findings | Fixed versions reported by scanner |
| --- | --- | --- |
| `google.golang.org/grpc` 1.80.0 | CVE-2026-84445, CVE-2026-84304, GHSA-hrxh-6v49-42gf | Initial reported thresholds: 1.82.2, 1.83.1, 1.82.1 |
| `golang.org/x/crypto` 0.53.0 | CVE-2026-56854, CVE-2026-56855, CVE-2026-78662 | 0.56.0 covers all three |
| `go.opentelemetry.io/otel` 1.39.0 | CVE-2026-29181 | 1.41.0 |

These are package-level findings, not proof of reachable exploitation in our
deployment. Updating this repository's plugin dependencies does not patch the
separate upstream controller executable. At this stage the combined image
still had high vulnerabilities; the September 15 rebuild above supersedes
that security gate.

## Plugin gRPC update, 2026-09-11

The plugin now pins gRPC **1.83.2**, with the required transitive updates to
`x/net` 0.58.0, `x/sys` 0.47.0, `x/term` 0.45.0 and `x/text` 0.41.0.
Go remains pinned to 1.26.8. `make test` passes, including the executable
handshake and local success, failure, malformed-response and RPC error tests.

The earlier apparent scanner discrepancy was an incorrect interpretation of
branch-specific fixes: CVE-2026-84445 is patched in **1.82.2 and 1.83.2**;
1.83.1 is still affected. See the
[upstream advisory](https://github.com/grpc/grpc-go/security/advisories/GHSA-2v4p-qf9q-27wj).
The plugin uses net/rpc, not an xDS gRPC server, but the dependency is upgraded
instead of retaining the affected version.

Validation on 2026-09-11: `go mod verify` passes. The local AMD64 image
`argo-rollouts-coralogix:TW-6901-grpc-1.83.2` builds and reports controller
version `v1.10.0+d90700a`. Docker Scout reports **zero critical and seven high
advisory IDs**, all located in `/bin/rollouts-controller`; none of the remaining
high/critical findings are attributed to the plugin executable. The plugin's
CVE-2026-84445 match is gone. This is a scanner result, not a claim that the
plugin is free of every vulnerability.

This change does not rebuild or patch the upstream controller or dashboard.
Their security assessment and staging activation gates remain open.

## Remaining rollout gates

1. Complete review/testing, publish both traceable patched artifacts, re-scan
   the published images and record their registry digests and source commit.
2. Require both patched image digests in the candidate chart, then activate
   only for `uc1-stg-sync` via a separate PR.
3. Run actual Coralogix analysis, promotion, abort/rollback, pause/resume,
   controller restart, traffic-routing, dashboard and CLI checks in staging.
4. Observe normal staging deployments before promoting the same artifact to
   production. Follow the chart's `UPGRADE.md` recovery and evidence checklist.
5. Upgrade central Argo CD and migrate IAP access as separate TW-6901 changes.

No cluster upgrade, IAP change, or Teleport removal is performed by this PR.
