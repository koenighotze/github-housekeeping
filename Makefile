.DEFAULT_GOAL := build

.PHONY: all build test test.all test.report vet fmt lint.local deps.upgrade deps.vulncheck.local clean get.dependencies

get.dependencies:
	go mod tidy

fmt:
	go fmt ./cmd/... ./internal/... ./pkg/...

vet: fmt
	go vet ./cmd/... ./internal/... ./pkg/...

lint.local:
	golangci-lint run ./...

install.tools.local:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	brew install golangci-lint

deps.upgrade:
	go get -u ./...
	go mod tidy

deps.vulncheck.local:
	govulncheck ./...

test: get.dependencies
	go test ./internal/... ./cmd/... ./pkg/... -coverprofile=coverage.out

test.all: get.dependencies
	go test -tags=integration ./internal/... ./cmd/... ./pkg/... -coverprofile=coverage.out

test.report: test
	go tool cover -html=coverage.out

build: get.dependencies
	go build -o github-housekeeping ./cmd/housekeeping

run.local:
	go run ./cmd/housekeeping/main.go

clean:
	go clean -x -i
	rm -f ./github-housekeeping
