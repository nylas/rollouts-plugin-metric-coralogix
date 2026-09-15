#!/bin/sh
set -eu

controller_image=${1:?controller image required}
dashboard_image=${2:?dashboard image required}
fixture_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
dashboard_id=
cleanup() {
    if [ -n "$dashboard_id" ]; then
        docker rm -f "$dashboard_id" >/dev/null
    fi
}
trap cleanup EXIT
trap 'exit 1' HUP INT TERM

for image in "$controller_image" "$dashboard_image"; do
    test "$(docker image inspect --format '{{.Config.User}}' "$image")" = 999
    test "$(docker image inspect --format '{{.Architecture}}' "$image")" = amd64
done
docker run --rm --network none --platform linux/amd64 "$controller_image" --version
docker run --rm --network none --platform linux/amd64 "$dashboard_image" version

# Only publish to loopback. No real kubeconfig or cloud credentials are mounted.
dashboard_id=$(docker run -d --platform linux/amd64 --read-only --cap-drop ALL \
    --security-opt no-new-privileges=true -p 127.0.0.1::3100 \
    --mount "type=bind,src=$fixture_dir/smoke-kubeconfig.yaml,dst=/test/kubeconfig,readonly" \
    "$dashboard_image" dashboard --kubeconfig /test/kubeconfig --namespace default)
address=$(docker port "$dashboard_id" 3100/tcp)
case "$address" in
    127.0.0.1:[0-9]*) ;;
    *) echo 'Unexpected dashboard binding' >&2; exit 1 ;;
esac
page=$(curl --fail --silent --show-error --retry 20 --retry-all-errors \
    --retry-delay 1 --retry-max-time 30 --max-time 5 "http://$address/rollouts/")
asset=$(printf '%s\n' "$page" | sed -nE 's/.*src="([^"]+\.js)".*/\1/p' | head -n 1)
case "$asset" in
    ''|*..*|*://*|//*) echo 'Missing or invalid embedded JavaScript asset' >&2; exit 1 ;;
esac
case "$asset" in
    /*) asset_url="http://$address$asset" ;;
    *) asset_url="http://$address/rollouts/$asset" ;;
esac
asset_type=$(curl --fail --silent --show-error --max-time 15 \
    --output /dev/null --write-out '%{content_type}' "$asset_url")
case "$asset_type" in
    text/javascript*|application/javascript*) ;;
    *) echo 'Embedded JavaScript returned an unexpected content type' >&2; exit 1 ;;
esac
# This read-only endpoint also exercises the HTTP-to-gRPC gateway without
# requiring a Kubernetes API, unlike rollout listing or mutation endpoints.
api_version=$(curl --fail --silent --show-error --max-time 15 \
    "http://$address/rollouts/api/v1/version" | jq -er '.rolloutsVersion')
test "$api_version" = 'v1.10.0+d90700a.dirty'
echo 'Image versions, UID, architecture, dashboard HTML, JavaScript and gRPC gateway smoke checks passed.'
