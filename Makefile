# gghstats — build, quality and release workflow

BINARY      := gghstats
MODULE      := github.com/hrodrig/gghstats
DIST        := dist
# Single source of truth: VERSION file at repo root (no silent fallback — avoids wrong port/tarball names).
VERSION_RAW ?= $(shell cat VERSION 2>/dev/null | tr -d '\n\r')
VERSION     := $(patsubst v%,%,$(VERSION_RAW))
TAG         := v$(VERSION)
COMMIT      := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BUILDDATE   := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
# Fails early when Docker is required but not running.
check-docker = @docker info >/dev/null 2>&1 || { echo "Error: Docker is not running. Start Docker and try again."; exit 1; }
GRYPE_FAIL_ON  ?= high
# OpenBSD dist helper default arch. Override: gmake dist-openbsd OPENBSD_ARCH=arm64
OPENBSD_ARCH   ?= amd64
# Empty = native arch. Set linux/amd64 when building on Apple Silicon (or arm64) for a typical VPS.
DOCKER_PLATFORM ?=
LDFLAGS     := -s -w \
	-X '$(MODULE)/internal/version.Version=$(VERSION)' \
	-X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
	-X '$(MODULE)/internal/version.BuildDate=$(BUILDDATE)'

.DEFAULT_GOAL := help

LIMIT ?=

.PHONY: build check-x-net-pin clean compose-down compose-up cover dist-freebsd dist-openbsd docker-build docker-build-amd64 docker-export-amd64 docker-run docker-scan gocyclo govulncheck grype help install install-man lint lint-fix port-freebsd-sync port-openbsd-sync release release-check security server server-metrics snapshot test test-platforms test-platforms-ping test-release tools

# Minimum golang.org/x/net (explicit go.mod pin; go mod tidy drops it → Prometheus resolves v0.43.0).
X_NET_MIN_VERSION ?= v0.57.0
# Project-wide statement coverage floor (make cover / release-check). SPEC §6.1.
COVER_MIN_PERCENT ?= 80

GREEN  := \033[0;32m
YELLOW := \033[0;33m
RESET  := \033[0m

help:
	@echo "$(GREEN)gghstats$(RESET) — GitHub traffic dashboard and CLI"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "$(YELLOW)Build:$(RESET)"
	@echo "  $(GREEN)build$(RESET)            Build local binary"
	@echo "  $(GREEN)clean$(RESET)            Remove local build artifacts"
	@echo "  $(GREEN)compose-down$(RESET)     Stop stack with docker compose"
	@echo "  $(GREEN)compose-up$(RESET)       Start stack with docker compose"
	@echo "  $(GREEN)install$(RESET)          Install binary with ldflags"
	@echo "  $(GREEN)install-man$(RESET)      Install man page (MANDIR=/usr/local/share/man by default)"
	@echo "  $(GREEN)server$(RESET)           Run gghstats serve locally (go run)"
	@echo "  $(GREEN)server-metrics$(RESET)   Same as server with GGHSTATS_METRICS_PER_REPO=true"
	@echo ""
	@echo "$(YELLOW)Quality:$(RESET)"
	@echo "  $(GREEN)cover$(RESET)            Run tests with coverage; fail if total < $(COVER_MIN_PERCENT)% (SPEC §6.1)"
	@echo "  $(GREEN)grype$(RESET)            Grype directory scan (excludes ./dist/**, ./gghstats)"
	@echo "  $(GREEN)lint$(RESET)             Check gofmt, go vet, and x/net security pin"
	@echo "  $(GREEN)lint-fix$(RESET)         Apply gofmt -s -w"
	@echo "  $(GREEN)check-x-net-pin$(RESET)  Verify golang.org/x/net pin in go.mod (see X_NET_MIN_VERSION)"
	@echo "  $(GREEN)security$(RESET)         Run govulncheck, gocyclo and grype"
	@echo "  $(GREEN)tools$(RESET)            Install govulncheck and gocyclo"
	@echo "  $(GREEN)test$(RESET)             Run unit tests"
	@echo "  $(GREEN)test-platforms$(RESET)   Ansible full-cycle on lab VMs (testing/platforms/; needs hosts.yml)"
	@echo "  $(GREEN)test-platforms-ping$(RESET)  SSH/Python connectivity check for platform inventory"
	@echo ""
	@echo "$(YELLOW)Docker:$(RESET)"
	@echo "  $(GREEN)docker-build$(RESET)       Build image gghstats:$(VERSION) (optional: DOCKER_PLATFORM=linux/amd64)"
	@echo "  $(GREEN)docker-build-amd64$(RESET) Same, forced linux/amd64 (VPS / x86_64 validation)"
	@echo "  $(GREEN)docker-export-amd64$(RESET) Build amd64 image and write dist/gghstats-$(VERSION)-linux-amd64.tar.gz for docker load on VPS"
	@echo "  $(GREEN)docker-run$(RESET)         Run local Docker image"
	@echo "  $(GREEN)docker-scan$(RESET)        Build and scan image with Grype (pass DOCKER_PLATFORM=... to match target arch)"
	@echo ""
	@echo "$(YELLOW)Release:$(RESET)"
	@echo "  $(GREEN)release$(RESET)            Publish release (main branch only)"
	@echo "  $(GREEN)release-check$(RESET)      lint, test, cover (≥$(COVER_MIN_PERCENT)%), security, docker-scan (Docker)"
	@echo "  $(GREEN)snapshot$(RESET)           Goreleaser snapshot (VERSION → <semver>-next, dist/, no Docker)"
	@echo "  $(GREEN)test-release$(RESET)       Snapshot dry-run (--skip=publish; same VERSION source)"
	@echo "  $(GREEN)dist-freebsd$(RESET)       Build FreeBSD tar.gz distfile for ports local testing"
	@echo "  $(GREEN)dist-openbsd$(RESET)       Build OpenBSD tar.gz distfile for ports local testing (OPENBSD_ARCH=amd64)"
	@echo "  $(GREEN)port-freebsd-sync$(RESET)  Sync VERSION to contrib/freebsd/Makefile (before port update)"
	@echo "  $(GREEN)port-openbsd-sync$(RESET)  Sync VERSION to contrib/openbsd/port/Makefile (before port update)"
	@echo ""
	@echo "Current version: $(VERSION) (tag: $(TAG))"
	@echo ""
	@echo "Examples:"
	@echo "  make test-platforms LIMIT=gghstats-ubuntu"

build:
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/gghstats

install:
	go install -ldflags "$(LDFLAGS)" ./cmd/gghstats

MANDIR ?= /usr/local/share/man
install-man:
	@mkdir -p $(MANDIR)/man1
	@cp contrib/man/man1/gghstats.1 $(MANDIR)/man1/
	@echo "Installed man page to $(MANDIR)/man1/gghstats.1"

server:
	go run -ldflags "$(LDFLAGS)" ./cmd/gghstats serve

# Local dev: expose per-repo Prometheus gauges (higher cardinality).
server-metrics:
	GGHSTATS_METRICS_PER_REPO=true go run -ldflags "$(LDFLAGS)" ./cmd/gghstats serve

compose-up:
	docker compose up -d

compose-down:
	docker compose down

test:
	go test -race ./...

.PHONY: test-platforms test-platforms-ping
test-platforms:
	@command -v ansible-playbook >/dev/null 2>&1 || { echo "ansible-playbook not found; install Ansible 2.14+"; exit 1; }
	@test -f testing/platforms/inventory/hosts.yml || { echo "Missing testing/platforms/inventory/hosts.yml — copy hosts.yml.example and edit."; exit 1; }
	cd testing/platforms && ansible-playbook playbooks/full-cycle.yml $(if $(LIMIT),--limit $(LIMIT),)

test-platforms-ping:
	@command -v ansible-playbook >/dev/null 2>&1 || { echo "ansible-playbook not found; install Ansible 2.14+"; exit 1; }
	@test -f testing/platforms/inventory/hosts.yml || { echo "Missing testing/platforms/inventory/hosts.yml — copy hosts.yml.example and edit."; exit 1; }
	cd testing/platforms && ansible-playbook playbooks/ping.yml $(if $(LIMIT),--limit $(LIMIT),)

cover:
	go test ./... -coverprofile=coverage.out -covermode=atomic
	@total=$$(go tool cover -func=coverage.out | tail -1); echo "$$total"; \
	pct=$$(echo "$$total" | awk '{print $$NF}' | tr -d '%'); \
	awk -v p="$$pct" -v min="$(COVER_MIN_PERCENT)" 'BEGIN { if ((p+0) < (min+0)) { printf "coverage %s%% must be >= %s%% (SPEC §6.1)\n", p, min; exit 1 } }'

lint:
	@$(MAKE) check-x-net-pin
	@echo "Checking gofmt -s..."
	@unformatted=$$(gofmt -s -l .); [ -z "$$unformatted" ] || { echo "Files not formatted (run make lint-fix):"; echo "$$unformatted"; exit 1; }
	@echo "Running go vet..."
	@go vet ./...

# Ensure the explicit x/net pin stays in go.mod — Snyk expects >= X_NET_MIN_VERSION.
# Auto-restores the pin if go mod tidy (or CI runner setup) dropped it.
check-x-net-pin:
	@echo "Checking golang.org/x/net pin (minimum $(X_NET_MIN_VERSION))..."
	@grep -q 'golang.org/x/net $(X_NET_MIN_VERSION)' go.mod || { \
		echo "go.mod missing pin; re-pinning golang.org/x/net@$(X_NET_MIN_VERSION)..."; \
		go get golang.org/x/net@$(X_NET_MIN_VERSION); \
	}
	@resolved=$$(go list -m -f '{{.Version}}' golang.org/x/net); \
	if [ "$$resolved" != "$(X_NET_MIN_VERSION)" ]; then \
		echo "golang.org/x/net resolved to $$resolved; re-pinning to $(X_NET_MIN_VERSION)..."; \
		go get golang.org/x/net@$(X_NET_MIN_VERSION); \
	fi
	@echo "golang.org/x/net pin OK ($(X_NET_MIN_VERSION))"

lint-fix:
	gofmt -s -w .

clean:
	rm -f $(BINARY)
	rm -rf $(DIST)

docker-build:
	$(check-docker)
	@set -e; \
	opts=""; \
	if [ -n "$(strip $(DOCKER_PLATFORM))" ]; then opts="--platform $(DOCKER_PLATFORM)"; fi; \
	DOCKER_BUILDKIT=1 docker build $$opts \
		--build-arg VERSION=$(VERSION) --build-arg COMMIT=$(COMMIT) --build-arg BUILDDATE=$(BUILDDATE) \
		-t gghstats:$(VERSION) .

docker-build-amd64:
	$(MAKE) docker-build DOCKER_PLATFORM=linux/amd64

docker-export-amd64: docker-build-amd64
	@mkdir -p $(DIST)
	docker save gghstats:$(VERSION) | gzip -c > $(DIST)/gghstats-$(VERSION)-linux-amd64.tar.gz
	@echo "Wrote $(DIST)/gghstats-$(VERSION)-linux-amd64.tar.gz"
	@echo "On VPS: gunzip -c gghstats-$(VERSION)-linux-amd64.tar.gz | docker load"

# Docker path: --pull=always so anchore/grype:latest is not a stale local cache (latest != auto-update).
docker-scan: docker-build
	@if command -v grype >/dev/null 2>&1; then \
		grype gghstats:$(VERSION) --fail-on $(GRYPE_FAIL_ON) ; \
	else \
		echo "grype not found locally, using container image..."; \
		docker run --rm --pull=always -v /var/run/docker.sock:/var/run/docker.sock anchore/grype:latest gghstats:$(VERSION) --fail-on $(GRYPE_FAIL_ON) ; \
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

# Dir scan: exclude local build outputs — binaries embed buildinfo (older Go) and skew stdlib CVE noise vs `go version`.
# Syft/Grype require exclusion globs to start with ./, */, or **/ (see grype catalog error).
GRYPE_DIR_EXCLUDES := --exclude './dist/**' --exclude './gghstats'

grype:
	@if command -v grype >/dev/null 2>&1; then \
		grype dir:. $(GRYPE_DIR_EXCLUDES) ; \
	else \
		echo "grype not found locally, using container image..."; \
		docker run --rm --pull=always -v "$(PWD):/workspace" anchore/grype:latest \
			dir:/workspace $(GRYPE_DIR_EXCLUDES) ; \
	fi

security: govulncheck gocyclo grype

# Sync VERSION file to FreeBSD port Makefile. Run before updating the port for a new release.
.PHONY: port-freebsd-sync
port-freebsd-sync:
	@[ -n "$(VERSION)" ] || { echo "Error: VERSION file empty or missing"; exit 1; }
	@sed -i.bak "s/^PORTVERSION=.*/PORTVERSION=\t$(VERSION)/" contrib/freebsd/Makefile
	@rm -f contrib/freebsd/Makefile.bak
	@echo "Updated contrib/freebsd/Makefile PORTVERSION to $(VERSION)"

# Build only the FreeBSD distfile tarball expected by contrib/freebsd/Makefile DISTFILES.
.PHONY: dist-freebsd
dist-freebsd:
	@test -f VERSION || { echo "Error: VERSION file missing at repo root"; exit 1; }
	@[ -n "$(VERSION)" ] || { echo "Error: VERSION is empty"; exit 1; }
	@ver="$(VERSION)"; \
	arch=$$(uname -m | sed 's/^aarch64$$/arm64/'); \
	out="$(DIST)/gghstats_$${ver}_freebsd_$$arch.tar.gz"; \
	stage="/tmp/gghstats-dist-root-$$PPID"; \
	set -e; \
	echo "Building gghstats for FreeBSD $$arch with VERSION=$$ver..."; \
	rm -rf "$$stage"; \
	mkdir -p "$$stage/share/man/man1" "$$stage/share/doc/gghstats" "$$stage/etc/gghstats" "$(DIST)"; \
	GOOS=freebsd GOARCH=$$arch CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o "$$stage/gghstats" ./cmd/gghstats; \
	cp contrib/man/man1/gghstats.1 "$$stage/share/man/man1/gghstats.1"; \
	cp LICENSE "$$stage/share/doc/gghstats/LICENSE"; \
	cp contrib/gghstats.env.example "$$stage/etc/gghstats/gghstats.env.example"; \
	tar -C "$$stage" -czf "$$out" .; \
	rm -rf "$$stage"; \
	echo "Wrote $$out"

# Sync VERSION file to OpenBSD port Makefile. Run before updating the port for a new release.
.PHONY: port-openbsd-sync
port-openbsd-sync:
	@[ -n "$(VERSION)" ] || { echo "Error: VERSION file empty or missing"; exit 1; }
	@test -f contrib/openbsd/port/Makefile || { echo "Error: contrib/openbsd/port/Makefile not found"; exit 1; }
	@sed -i.bak \
	  -e 's#^DISTNAME =.*#DISTNAME =	gghstats_$(VERSION)_openbsd_$${MACHINE_ARCH:S/aarch64/arm64/}#' \
	  -e 's#^PKGNAME =.*#PKGNAME =	gghstats-$(VERSION)#' \
	  -e 's#^MASTER_SITES =.*#MASTER_SITES =	https://github.com/hrodrig/gghstats/releases/download/v$(VERSION)/#' \
	  -e 's#^DISTFILES =.*#DISTFILES =	gghstats_$(VERSION)_openbsd_$${MACHINE_ARCH:S/aarch64/arm64/}.tar.gz#' \
	  contrib/openbsd/port/Makefile
	@rm -f contrib/openbsd/port/Makefile.bak
	@cp contrib/openbsd/gghstats contrib/openbsd/port/pkg/gghstats.rc
	@cp contrib/openbsd/gghstats contrib/openbsd/port/files/gghstats
	@cp contrib/openbsd/gghstats-serve contrib/openbsd/port/files/gghstats-serve
	@cp contrib/openbsd/gghstats-start contrib/openbsd/port/files/gghstats-start
	@echo "Updated contrib/openbsd/port/Makefile to $(VERSION)"
	@echo "Synced contrib/openbsd/port/files/ from contrib/openbsd/"

# Build only the OpenBSD distfile tarball expected by contrib/openbsd/port/Makefile DISTFILES.
.PHONY: dist-openbsd
dist-openbsd:
	@test -f VERSION || { echo "Error: VERSION file missing at repo root"; exit 1; }
	@[ -n "$(VERSION)" ] || { echo "Error: VERSION is empty"; exit 1; }
	@echo "$(OPENBSD_ARCH)" | grep -qE '^(amd64|arm64|riscv64)$$' || { echo "Error: OPENBSD_ARCH must be one of: amd64, arm64, riscv64"; exit 1; }
	@ver="$(VERSION)"; \
	arch="$(OPENBSD_ARCH)"; \
	out="$(DIST)/gghstats_$${ver}_openbsd_$$arch.tar.gz"; \
	stage="/tmp/gghstats-openbsd-dist-root-$$PPID"; \
	set -e; \
	echo "Building gghstats for OpenBSD $$arch with VERSION=$$ver..."; \
	rm -rf "$$stage"; \
	mkdir -p "$$stage/share/man/man1" "$$stage/share/doc/gghstats" "$$stage/etc/gghstats" "$$stage/share/openbsd/rc.d" "$(DIST)"; \
	GOOS=openbsd GOARCH=$$arch CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o "$$stage/gghstats" ./cmd/gghstats; \
	cp contrib/man/man1/gghstats.1 "$$stage/share/man/man1/gghstats.1"; \
	cp LICENSE "$$stage/share/doc/gghstats/LICENSE"; \
	cp contrib/gghstats.env.example "$$stage/etc/gghstats/gghstats.env.example"; \
	cp contrib/openbsd/gghstats "$$stage/share/openbsd/rc.d/gghstats"; \
	cp contrib/openbsd/gghstats-serve "$$stage/gghstats-serve"; \
	cp contrib/openbsd/gghstats-start "$$stage/gghstats-start"; \
	tar -C "$$stage" -czf "$$out" .; \
	rm -rf "$$stage"; \
	echo "Wrote $$out"

release-check:
	$(check-docker)
	@test -f VERSION || (echo "VERSION file is required"; exit 1)
	@echo "Release version: $(VERSION) (tag: $(TAG))"
	@echo "$(VERSION)" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || (echo "VERSION must be semantic version (e.g. 0.1.0)"; exit 1)
	@command -v goreleaser >/dev/null 2>&1 || (echo "goreleaser is required. Install from https://goreleaser.com/install/"; exit 1)
	@echo "Running release checks (lint, test, cover ≥$(COVER_MIN_PERCENT)%, security, docker-scan)..."
	@$(MAKE) lint
	@$(MAKE) test
	@$(MAKE) cover
	@$(MAKE) security
	@$(MAKE) docker-scan
	@echo "All release checks passed."

# Snapshot version from VERSION (e.g. 0.5.0 => 0.5.0-next), independent of latest git tag.
define export_gghstats_snapshot_version
	set -e; \
	ver_raw=$$(cat VERSION 2>/dev/null | tr -d '\n\r'); \
	[ -n "$$ver_raw" ] || { echo "Error: VERSION file is required"; exit 1; }; \
	ver=$${ver_raw#v}; \
	echo "$$ver" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+$$' || { echo "Error: VERSION must be semantic MAJOR.MINOR.PATCH (got: $$ver_raw)"; exit 1; }; \
	export GGHSTATS_SNAPSHOT_VERSION="$$ver-next"; \
	echo "Snapshot version: $$GGHSTATS_SNAPSHOT_VERSION (from VERSION)"
endef

snapshot: release-check
	@$(export_gghstats_snapshot_version); \
	goreleaser release --snapshot --clean --skip=homebrew

test-release: release-check
	@$(export_gghstats_snapshot_version); \
	goreleaser release --snapshot --skip=publish --clean --skip=homebrew

release: release-check
	@branch=$$(git branch --show-current 2>/dev/null); \
	if [ "$$branch" != "main" ]; then \
	  echo "Error: release only from main (current: $$branch)."; \
	  exit 1; \
	fi
	goreleaser release --clean
