# TW-6901: Rollouts v1.10.0 readiness

Tracking: https://nylas.atlassian.net/browse/TW-6901

Controller/plugin PR: https://github.com/nylas/rollouts-plugin-metric-coralogix/pull/8

The paired `unicorn-helm` candidate chart is
`chart/argo-rollouts-1.10/chart` (upstream chart 2.43.1). It requires the published
controller image digest and is not referenced by any active ApplicationSet.

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
separate upstream controller executable. Keep the PR in draft and resolve or
record the approved reachability/risk assessment before staging activation;
do not report the combined image as free of high vulnerabilities.

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

1. Complete the upstream-controller security assessment, then publish a
   traceable controller artifact and record its final digest and source commit.
2. Activate the candidate chart only for `uc1-stg-sync` via a separate PR.
3. Run actual Coralogix analysis, promotion, abort/rollback, pause/resume,
   controller restart, traffic-routing, dashboard and CLI checks in staging.
4. Observe normal staging deployments before promoting the same artifact to
   production. Follow the chart's `UPGRADE.md` recovery and evidence checklist.
5. Upgrade central Argo CD and migrate IAP access as separate TW-6901 changes.

No cluster upgrade, IAP change, or Teleport removal is performed by this PR.
