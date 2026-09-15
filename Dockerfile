# syntax=docker/dockerfile:1
# TW-6901: rebuild both upstream executables, not only the metric plugin.
FROM --platform=$BUILDPLATFORM golang:1.26.8-alpine@sha256:ce864e7223ac17b1775e6fd0b4c0db580c2eb50e7953a427916379e4b92a1628 AS go-builder
RUN apk add --no-cache make git
ENV GOTOOLCHAIN=local

FROM go-builder AS plugin-build
WORKDIR /plugin
COPY go.mod go.sum ./
RUN go mod download && go mod verify
COPY main.go ./
COPY internal ./internal
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -mod=readonly -trimpath \
    -ldflags="-s -w" -o /out/rollouts-plugin-metric-coralogix-linux-amd64 .

FROM go-builder AS upstream-source
WORKDIR /src
RUN git init && git remote add origin https://github.com/argoproj/argo-rollouts.git && \
    git fetch --depth=1 origin d90700ae8d71d141561f0c546e19f999bb335cbd && \
    git checkout --detach FETCH_HEAD && \
    test "$(git rev-parse HEAD)" = d90700ae8d71d141561f0c546e19f999bb335cbd

FROM upstream-source AS upstream-deps
COPY build/argo-rollouts/go.mod build/argo-rollouts/go.sum ./
RUN go mod download && go mod verify
ENV GOFLAGS="-mod=readonly -trimpath"

FROM upstream-deps AS controller-build
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=linux GOARCH=amd64 make controller VERSION=1.10.0 GIT_TAG= \
    GIT_COMMIT=d90700ae8d71d141561f0c546e19f999bb335cbd GIT_TREE_STATE=dirty \
    BUILD_DATE=2026-09-15T00:00:00Z LDFLAGS="-s -w"

FROM --platform=$BUILDPLATFORM node:22-bookworm-slim@sha256:83f487e0a63425e5b4d146fb5e5be574bcbe1b7b843d3ebafdd95eaf7767a7e5 AS dashboard-ui
RUN apt-get update && apt-get install -y --no-install-recommends git ca-certificates && \
    npm install --global pnpm@11.9.0
WORKDIR /src/ui
COPY --from=upstream-source /src/ui/ ./
RUN pnpm install --frozen-lockfile
RUN ARGO_VERSION=v1.10.0 NODE_ENV=production pnpm run build

FROM upstream-deps AS dashboard-build
COPY --from=dashboard-ui /src/ui/dist/app/ ./ui/dist/app/
RUN --mount=type=cache,target=/root/.cache/go-build \
    GOOS=linux GOARCH=amd64 make plugin VERSION=1.10.0 GIT_TAG= \
    GIT_COMMIT=d90700ae8d71d141561f0c546e19f999bb335cbd GIT_TREE_STATE=dirty \
    BUILD_DATE=2026-09-15T00:00:00Z LDFLAGS="-s -w"

# Native-platform unit tests avoid emulating an AMD64 test process on ARM hosts.
FROM upstream-deps AS upstream-test
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go test -short -count=1 -p 2 ./controller/... ./rollout/... ./metricproviders/... ./server/...

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab AS runtime
LABEL org.opencontainers.image.source="https://github.com/nylas/rollouts-plugin-metric-coralogix" \
    org.opencontainers.image.version="v1.10.0-nylas-TW-6901" \
    com.nylas.argo-rollouts.upstream-revision="d90700ae8d71d141561f0c546e19f999bb335cbd"
USER 999
WORKDIR /home/argo-rollouts

FROM runtime AS dashboard
COPY --from=dashboard-build /src/dist/kubectl-argo-rollouts /bin/kubectl-argo-rollouts
ENTRYPOINT ["/bin/kubectl-argo-rollouts"]
CMD ["dashboard"]

# Keep the default build target compatible with the existing controller image.
FROM runtime AS controller
COPY --from=controller-build /src/dist/rollouts-controller /bin/rollouts-controller
COPY --from=plugin-build /out/rollouts-plugin-metric-coralogix-linux-amd64 /home/plugins-temp/rollouts-plugin-metric-coralogix-linux-amd64
ENTRYPOINT ["/bin/rollouts-controller"]
