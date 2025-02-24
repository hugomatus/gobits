# Go parameters
GOCMD=go
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
COVERAGE_FILE=coverage.out

# Force flags
FORCE_FLAGS=-count=1 -p=1
RACE_FLAGS=-race

# Source files
SOURCES=$(shell find . -name '*.go' -not -path "./vendor/*")

# Test coverage exclusions
COVERAGE_EXCLUSIONS=github.com/hugomatus/gobits/examples

# Conventional commit types
COMMIT_TYPES = fix feat docs style refactor perf test build ci chore revert
COMMIT_SCOPE = $(shell git status --porcelain | cut -d' ' -f2 | xargs dirname | sort -u)

.PHONY: all clean test coverage lint deps vendor help commit

all: clean deps fmt lint test coverage ## Run all checks with fresh builds

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

deps: ## Get dependencies
	$(GOGET) -t -v -u ./...
	$(GOMOD) tidy

vendor: ## Vendor dependencies
	$(GOMOD) vendor
	$(GOMOD) verify

clean: ## Remove build artifacts
	$(GOCMD) clean -cache -testcache
	rm -f $(COVERAGE_FILE)
	rm -rf dist/

test: ## Run tests without caching
	$(GOTEST) $(FORCE_FLAGS) $(RACE_FLAGS) -coverpkg=$(shell go list ./... | grep -v $(COVERAGE_EXCLUSIONS) | tr '\n' ',') ./...

coverage: ## Run tests with coverage without caching
	$(GOTEST) $(FORCE_FLAGS) $(RACE_FLAGS) -coverprofile=$(COVERAGE_FILE) -covermode=atomic -coverpkg=$(shell go list ./... | grep -v $(COVERAGE_EXCLUSIONS) | tr '\n' ',') ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE)

coverage-text: ## Show coverage in terminal without caching
	$(GOTEST) $(FORCE_FLAGS) $(RACE_FLAGS) -coverprofile=$(COVERAGE_FILE) -covermode=atomic -coverpkg=$(shell go list ./... | grep -v $(COVERAGE_EXCLUSIONS) | tr '\n' ',') ./...
	$(GOCMD) tool cover -func=$(COVERAGE_FILE)

lint: ## Run linters
	golangci-lint cache clean
	golangci-lint run ./...

benchmark: ## Run benchmarks without caching
	$(GOTEST) $(FORCE_FLAGS) -bench=. -benchmem ./...

check: lint test ## Run linters and tests without caching

ci: clean deps fmt-check check coverage-text ## Run all CI steps with fresh builds

# Install development tools
tools: ## Install development tools
	go install -v github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install -v golang.org/x/tools/cmd/goimports@latest
	go install -v github.com/golang/mock/mockgen@latest

fmt: ## Format code
	@echo "==> Formatting code"
	@gofmt -l -w $(SOURCES)
	@goimports -w $(SOURCES)

fmt-check: ## Check if code is formatted
	@echo "==> Checking code format"
	@test -z $(shell gofmt -l $(SOURCES))

quality: clean fmt-check fmt lint test coverage ## Run all quality checks with fresh builds

commit: ## Create a commit following Conventional Commits standard
	@if [ -z "$(shell git status --porcelain)" ]; then \
		echo "No changes to commit"; \
		exit 1; \
	fi
	@echo "Staged files:"
	@git status --porcelain
	@echo "\nSelect commit type:"
	@select type in $(COMMIT_TYPES); do \
		if [ -n "$$type" ]; then \
			echo "\nAvailable scopes:"; \
			echo "$(COMMIT_SCOPE)"; \
			echo "\nEnter scope (optional, press enter to skip):"; \
			read scope; \
			echo "\nEnter commit message:"; \
			read message; \
			if [ -n "$$scope" ]; then \
				git commit -m "$$type($$scope): $$message"; \
			else \
				git commit -m "$$type: $$message"; \
			fi; \
			break; \
		fi; \
	done

# Default target
.DEFAULT_GOAL := help
