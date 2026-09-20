# tf-blast Makefile
# Ultra-fast, zero-trust blast radius analyzer for Terraform and OpenTofu

BINARY_NAME := tf-blast
CMD_DIR     := ./cmd/tf-blast
BUILD_DIR   := dist
GO          ?= go

VERSION     ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE        ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS     := -s -w -X main.Version=$(VERSION) -X main.CommitSHA=$(COMMIT)

.PHONY: all help build install clean test test-unit test-cli bench fmt lint tidy build-all test-fixtures

.DEFAULT_GOAL := help

## help: Display this list of available commands
help:
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/  /'

## build: Compile local tf-blast binary
build:
	$(GO) build -ldflags="$(LDFLAGS)" -o $(BINARY_NAME) $(CMD_DIR)

## install: Install binary to GOPATH/bin
install:
	$(GO) install -ldflags="$(LDFLAGS)" $(CMD_DIR)

## test: Run all unit, integration, and performance tests
test:
	$(GO) test -v ./...

## test-unit: Run unit tests only (pkg/...)
test-unit:
	$(GO) test -v ./pkg/...

## test-cli: Run CLI integration tests (cmd/tf-blast/...)
test-cli:
	$(GO) test -v ./cmd/tf-blast/...

## bench: Run sub-500ms performance benchmark on 1,000+ resources
bench:
	$(GO) test -v -bench=. .

## test-fixtures: Build and test against sample plans (clean, replacement, stateful-destroy)
test-fixtures: build
	@echo "==> Testing clean plan (Low risk)..."
	./$(BINARY_NAME) testdata/clean-plan.json
	@echo ""
	@echo "==> Testing replacement plan (High risk)..."
	./$(BINARY_NAME) testdata/replacement-plan.json
	@echo ""
	@echo "==> Testing stateful destroy plan (Critical risk)..."
	./$(BINARY_NAME) testdata/stateful-destroy.json

## fmt: Format all Go source files
fmt:
	gofmt -s -w .

## lint: Run go vet and verify formatting
lint:
	$(GO) vet ./...
	@test -z "$$(gofmt -l .)" || (echo "Unformatted files found. Run 'make fmt'"; exit 1)

## tidy: Run go mod tidy and verify dependencies
tidy:
	$(GO) mod tidy

## vulncheck: Run govulncheck for known Go security vulnerabilities
vulncheck:
	$(GO) run golang.org/x/vuln/cmd/govulncheck@latest ./...

## build-all: Cross-compile binaries for Linux, macOS, and Windows into dist/
build-all: clean
	@mkdir -p $(BUILD_DIR)
	@echo "Building for darwin/arm64..."
	GOOS=darwin GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	@echo "Building for darwin/amd64..."
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	@echo "Building for linux/amd64..."
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	@echo "Building for linux/arm64..."
	GOOS=linux GOARCH=arm64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	@echo "Building for windows/amd64..."
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags="$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(CMD_DIR)
	@echo "Binaries created in $(BUILD_DIR)/"

## clean: Remove build artifacts and temporary files
clean:
	rm -rf $(BINARY_NAME) $(BUILD_DIR) pr-comment.md
