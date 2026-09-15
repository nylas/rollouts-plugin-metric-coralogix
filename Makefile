.PHONY: build-rollouts-plugin-metric-coralogix-debug
build-rollouts-plugin-metric-coralogix-debug:
	CGO_ENABLED=0 go build -gcflags="all=-N -l" -o rollouts-plugin-metric-coralogix main.go

.PHONY: build-rollouts-plugin-metric-coralogix
build-rollouts-plugin-metric-coralogix:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o rollouts-plugin-metric-coralogix-linux-amd64 main.go
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o rollouts-plugin-metric-coralogix-linux-arm64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w" -o rollouts-plugin-metric-coralogix-darwin-amd64 main.go
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w" -o rollouts-plugin-metric-coralogix-darwin-arm64 main.go

.PHONY: test
test:
	go test -v ./...

# TW-6901: local builds only; publishing and deployment are separate steps.
CONTROLLER_IMAGE ?= argo-rollouts-coralogix:TW-6901-security
DASHBOARD_IMAGE ?= argo-rollouts-dashboard:TW-6901-security

.PHONY: image-controller image-dashboard test-upstream scan-images smoke-images
image-controller:
	docker buildx build --platform linux/amd64 --target controller --load -t $(CONTROLLER_IMAGE) .

image-dashboard:
	docker buildx build --platform linux/amd64 --target dashboard --load -t $(DASHBOARD_IMAGE) .

test-upstream:
	docker buildx build --target upstream-test .

scan-images:
	docker scout cves --exit-code --only-severity critical,high $(CONTROLLER_IMAGE)
	docker scout cves --exit-code --only-severity critical,high $(DASHBOARD_IMAGE)

smoke-images:
	sh build/smoke-images.sh $(CONTROLLER_IMAGE) $(DASHBOARD_IMAGE)

.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out
