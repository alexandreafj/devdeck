BINARY          := devdeck
CMD             := ./cmd/devdeck
BIN_DIR         := bin
COVERAGE_FILE   := coverage.out
COVER_THRESHOLD := 80
GOLANGCI_VERSION := v1.64.8

# Where `go install` places binaries: GOBIN if set, else GOPATH/bin.
GOBIN := $(shell go env GOBIN)
ifeq ($(GOBIN),)
GOBIN := $(shell go env GOPATH)/bin
endif
SHELL_RC := $(HOME)/.zshrc

.DEFAULT_GOAL := help

.PHONY: help
help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "} {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build the binary into ./bin
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/$(BINARY) $(CMD)
	@echo "built $(BIN_DIR)/$(BINARY)"

.PHONY: run
run: ## Run the app from source
	go run $(CMD)

.PHONY: test
test: ## Run unit tests
	go test ./...

.PHONY: test-race
test-race: ## Run tests with the race detector
	go test -race ./...

.PHONY: cover
cover: ## Run tests with coverage and open the HTML report
	go test -covermode=atomic -coverprofile=$(COVERAGE_FILE) ./...
	@go tool cover -func=$(COVERAGE_FILE) | tail -1
	go tool cover -html=$(COVERAGE_FILE)

.PHONY: cover-check
cover-check: ## Fail if total coverage is below COVER_THRESHOLD (mirrors CI)
	@go test -covermode=atomic -coverprofile=$(COVERAGE_FILE) ./... >/dev/null
	@total=$$(go tool cover -func=$(COVERAGE_FILE) | awk '/total:/ {print substr($$3, 1, length($$3)-1)}'); \
	echo "total coverage: $$total% (threshold $(COVER_THRESHOLD)%)"; \
	awk "BEGIN { exit !($$total >= $(COVER_THRESHOLD)) }" || { echo "FAIL: coverage below $(COVER_THRESHOLD)%"; exit 1; }

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: lint
lint: ## Run golangci-lint (installs it if missing)
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "installing golangci-lint $(GOLANGCI_VERSION)..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@$(GOLANGCI_VERSION); }
	golangci-lint run ./...

.PHONY: fmt
fmt: ## Format all Go source
	gofmt -w .

.PHONY: tidy
tidy: ## Tidy go.mod / go.sum
	go mod tidy

.PHONY: ci
ci: vet lint test-race cover-check ## Run the full local pipeline (mirrors CI)

.PHONY: install
install: ## Install devdeck onto your PATH (auto-configures ~/.zshrc if needed)
	go install $(CMD)
	@echo "installed $(BINARY) -> $(GOBIN)/$(BINARY)"
	@case ":$$PATH:" in \
		*":$(GOBIN):"*) echo "$(GOBIN) already on PATH ✓";; \
		*) $(MAKE) --no-print-directory _add-path;; \
	esac

.PHONY: _add-path
_add-path: # internal: append GOBIN to PATH in the shell rc, idempotently
	@if grep -q 'devdeck PATH' $(SHELL_RC) 2>/dev/null; then \
		echo "$(SHELL_RC) already configures the PATH ✓"; \
	else \
		printf '\n# >>> devdeck PATH >>>\nexport PATH="$$PATH:%s"\n# <<< devdeck PATH <<<\n' "$(GOBIN)" >> $(SHELL_RC); \
		echo "added $(GOBIN) to PATH in $(SHELL_RC)"; \
		echo "run 'source $(SHELL_RC)' or restart your shell, then: devdeck"; \
	fi

.PHONY: uninstall
uninstall: ## Remove the installed binary
	@rm -f $(GOBIN)/$(BINARY) && echo "removed $(GOBIN)/$(BINARY)"

.PHONY: clean
clean: ## Remove build and coverage artifacts
	rm -rf $(BIN_DIR) $(COVERAGE_FILE)
