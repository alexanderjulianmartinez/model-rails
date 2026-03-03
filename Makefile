SHELL := /bin/bash

GO ?= go
GOFMT ?= gofmt
GOFILES := $(shell git ls-files '*.go')

.PHONY: help fmt fmt-check lint test tidy build ci

help:
	@echo "Targets:"
	@echo "  fmt        Format Go files"
	@echo "  fmt-check  Fail if files need gofmt"
	@echo "  lint       Run golangci-lint"
	@echo "  test       Run tests"
	@echo "  tidy       Run go mod tidy"
	@echo "  build      Build all packages"
	@echo "  ci         Run fmt-check, lint, test"

fmt:
	@$(GOFMT) -w $(GOFILES)

fmt-check:
	@unformatted=$$($(GOFMT) -l $(GOFILES)); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed on:"; \
		echo "$$unformatted"; \
		exit 1; \
	fi

lint:
	@golangci-lint run

test:
	@$(GO) test ./...

tidy:
	@$(GO) mod tidy

build:
	@$(GO) build ./...

ci: fmt-check lint test
