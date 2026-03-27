# gghstats — build, quality and release workflow

BINARY      := gghstats
MODULE      := github.com/hrodrig/gghstats
DIST        := dist
VERSION_RAW ?= $(shell v=$$(cat VERSION 2>/dev/null | tr -d '\n\r'); [ -n "$$v" ] && echo "$$v" || echo "0.1.0")
VERSION     := $(patsubst v%,%,$(VERSION_RAW))
TAG         := v$(VERSION)
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILDDATE   := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
STRICT_RELEASE ?= 0
GRYPE_FAIL_ON  ?= high
LDFLAGS     := -s -w \
	-X '$(MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/version.BuildDate=$(BUILDDATE)'

.DEFAULT_GOAL := help

.PHONY: help build install server compose-up compose-down test lint lint-fix clean docker-build docker-scan docker-run tools govulncheck gocyclo grype security release-check snapshot test-release release

help:
	@echo "gghstats — GitHub traffic dashboard and CLI"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Build:"
	@echo "  build              Build local binary"
	@echo "  install            Install binary with ldflags"
	@echo "  server             Run gghstats serve locally (go run)"
	@echo "  compose-up         Start stack with docker compose"
	@echo "  compose-down       Stop stack with docker compose"
	@echo "  clean              Remove local build artifacts"
	@echo ""
	@echo "Quality:"
	@echo "  test               Run unit tests"
	@echo "  lint               Check gofmt and go vet"
	@echo "  lint-fix           Apply gofmt -s -w"
	@echo "  tools              Install govulncheck and gocyclo"
	@echo "  security           Run govulncheck, gocyclo and grype"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build       Build Docker image gghstats:$(VERSION)"
	@echo "  docker-scan        Build and scan image with Grype"
	@echo "  docker-run         Run local Docker image"
	@echo ""
	@echo "Release:"
	@echo "  release-check      Validate semver, tooling, lint, test and security"
	@echo "  snapshot           Goreleaser snapshot build (local artifacts)"
	@echo "  test-release       Simulate release without publishing"
	@echo "  release            Publish release (main branch only)"
	@echo ""
	@echo "Current version: $(VERSION) (tag: $(TAG))"

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/gghstats

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/gghstats

server:
	go run -ldflags "$(LDFLAGS)" ./cmd/gghstats serve

compose-up:
	docker compose up -d

compose-down:
	docker compose down

test:
	go test -race ./...

lint:
	@echo "Checking gofmt -s..."
	@unformatted=$$(gofmt -s -l .); [ -z "$$unformatted" ] || { echo "Files not formatted (run make lint-fix):"; echo "$$unformatted"; exit 1; }
	@echo "Running go vet..."
	@go vet ./...

lint-fix:
	gofmt -s -w .

clean:
	rm -f $(BINARY)
	rm -rf $(DIST)

docker-build:
	docker build --build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILDDATE=$(BUILDDATE) -t gghstats:$(VERSION) .

docker-scan: docker-build
	@if command -v grype >/dev/null 2>&1; then \
		grype gghstats:$(VERSION) --fail-on $(GRYPE_FAIL_ON) ; \
	else \
		echo "grype not found locally, using container image..."; \
		docker run --rm -v /var/run/docker.sock:/var/run/docker.sock anchore/grype:latest gghstats:$(VERSION) --fail-on $(GRYPE_FAIL_ON) ; \
	fi

docker-run:
	docker run --rm -p 8080:8080 --env-file .env -v $(PWD)/data:/data gghstats:$(VERSION)

tools:
	go install golang.org/x/vuln/cmd/govulncheck@latest
	go install github.com/fzipp/gocyclo/cmd/gocyclo@latest

govulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

gocyclo:
	@command -v gocyclo >/dev/null 2>&1 || go install github.com/fzipp/gocyclo/cmd/gocyclo@latest
	gocyclo -over 15 .

grype:
	@if command -v grype >/dev/null 2>&1; then \
		grype dir:. ; \
	else \
		echo "grype not found locally, using container image..."; \
		docker run --rm -v "$(PWD):/workspace" anchore/grype:latest dir:/workspace ; \
	fi

security: govulncheck gocyclo grype

release-check:
	@test -f VERSION || (echo "VERSION file is required"; exit 1)
	@echo "Release version: $(VERSION) (tag: $(TAG))"
	@echo "$(VERSION)" | rg '^[0-9]+\.[0-9]+\.[0-9]+$$' >/dev/null || (echo "VERSION must be semantic version (e.g. 0.1.0)"; exit 1)
	@command -v goreleaser >/dev/null 2>&1 || (echo "goreleaser is required. Install from https://goreleaser.com/install/"; exit 1)
	@$(MAKE) lint
	@$(MAKE) test
	@$(MAKE) security
	@if [ "$(STRICT_RELEASE)" = "1" ]; then \
		echo "STRICT_RELEASE=1 -> running docker-scan"; \
		$(MAKE) docker-scan; \
	else \
		echo "STRICT_RELEASE=0 -> skipping docker-scan"; \
	fi
	@echo "All release checks passed."

snapshot: release-check
	goreleaser build --snapshot --clean

test-release: release-check
	goreleaser release --snapshot --skip=publish --clean

release: release-check
	@branch=$$(git branch --show-current 2>/dev/null); \
	if [ "$$branch" != "main" ]; then \
	  echo "Error: release only from main (current: $$branch)."; \
	  exit 1; \
	fi
	goreleaser release --clean
