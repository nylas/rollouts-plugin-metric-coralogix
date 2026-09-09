# Build stage
FROM --platform=$BUILDPLATFORM golang:1.26.1-alpine@sha256:2389ebfa5b7f43eeafbd6be0c3700cc46690ef842ad962f6c5bd6be49ed82039 AS builder

# Install build dependencies
RUN apk add --no-cache make git

# Set the working directory
WORKDIR /build

# Copy the source code
COPY . .

# Download dependencies
RUN go mod download

# Build the plugin for AMD64
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o rollouts-plugin-metric-coralogix-linux-amd64 main.go && \
    chmod +x rollouts-plugin-metric-coralogix-linux-amd64

# Create plugins-temp directory in the builder
RUN mkdir -p /tmp/home/plugins-temp
RUN cp rollouts-plugin-metric-coralogix-linux-amd64 /tmp/home/plugins-temp/

# Final stage
FROM quay.io/argoproj/argo-rollouts:v1.10.0@sha256:187630ba722846bb8e079630364d3c29c8246a256be95785fd957b088f621c4e

# Copy the plugin directory structure from builder
COPY --from=builder /tmp/home /home

# The controller copies the configured plugin into its writable plugin-bin volume.
