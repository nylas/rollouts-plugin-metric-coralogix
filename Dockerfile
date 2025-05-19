# Build stage
FROM golang:1.21-alpine AS builder

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

# Final stage
FROM quay.io/argoproj/argo-rollouts:v1.8.2

# Copy the plugin binary with permissions
COPY --from=builder --chmod=755 /build/rollouts-plugin-metric-coralogix-linux-amd64 /home/argo-rollouts/plugin-bin/ 