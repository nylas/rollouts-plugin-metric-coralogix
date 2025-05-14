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

.PHONY: test-coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

