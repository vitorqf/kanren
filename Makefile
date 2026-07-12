BINARY := kanren
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build test lint tidy ci clean run

build: ## Build the binary
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/kanren

test: ## Run tests with race detector + coverage
	go test -race -coverprofile=coverage.out ./...

lint: ## Static analysis (vet + golangci-lint if present)
	go vet ./...
	@command -v golangci-lint >/dev/null 2>&1 && golangci-lint run || echo "golangci-lint not installed; ran go vet only"

tidy: ## Verify go.mod is tidy
	go mod tidy
	git diff --exit-code go.mod go.sum 2>/dev/null || (echo "go.mod/go.sum not tidy"; exit 1)

ci: build test lint ## What CI runs

clean:
	rm -rf bin coverage.out

run: build ## Build and run
	./bin/$(BINARY) $(ARGS)
