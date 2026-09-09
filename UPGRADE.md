# TW-6901: Rollouts v1.10.0 readiness

Tracking: https://nylas.atlassian.net/browse/TW-6901

Controller/plugin PR: https://github.com/nylas/rollouts-plugin-metric-coralogix/pull/8

The paired `unicorn-helm` candidate chart is
`chart/argo-rollouts-1.10/chart` (upstream chart 2.43.1). It requires the published
controller image digest and is not referenced by any active ApplicationSet.

## Security review, 2026-09-09

The initial custom image scan identified a vulnerable Go 1.26.1 standard library
in the plugin binary. The plugin build now targets patched Go 1.26.8 and gRPC
1.83.1. The rebuilt image passes local tests and container version checks. Its
final Docker Scout scan reports no critical findings and seven high advisory
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
separate upstream controller executable. The final scan also associates
CVE-2026-84445 with the plugin's gRPC 1.83.1, despite the initial reported fixed
threshold of 1.82.2. Reconcile that scanner/advisory discrepancy before declaring
the plugin free of high findings. Keep the PR in draft and resolve or
record the approved reachability/risk assessment before staging activation;
do not report the combined image as free of high vulnerabilities.

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
