# Root Makefile for go-infrastructure
#
# Per-service Makefiles (kris-<svc>/Makefile) are the source of truth.
# This file only orchestrates: invoke from CI and from your shell prompt
# without hard-coding paths.
#
# Common targets:
#   make build-all       build every kris-* service
#   make test-all        test every module (race + count=1)
#   make gen-all         regenerate proto / wire / go mod tidy per service
#   make lint            golangci-lint across the workspace
#   make fmt             gofumpt + goimports
#   make new-service NAME=foo GRPC=51000 HTTP=8200 OTHER=9200
#   make dev-deps-up     docker-compose up local mongo/redis/mysql/prom/grafana
#   make dev-deps-down

SERVICES := alpha beta gamma

# ---- Per-service forwarders ----
.PHONY: $(SERVICES:%=build-%) $(SERVICES:%=test-%) $(SERVICES:%=gen-%) $(SERVICES:%=run-%)
$(SERVICES:%=build-%):
	$(MAKE) -C kris-$(@:build-%=%) build
$(SERVICES:%=test-%):
	$(MAKE) -C kris-$(@:test-%=%) test
$(SERVICES:%=gen-%):
	$(MAKE) -C kris-$(@:gen-%=%) all
$(SERVICES:%=run-%):
	$(MAKE) -C kris-$(@:run-%=%) run

# ---- All-services ----
.PHONY: build-all
# build all kris-* services
build-all:
	@for s in $(SERVICES); do \
	  echo ">>> build $$s"; \
	  $(MAKE) -C kris-$$s build || exit 1; \
	done

.PHONY: test-all
# test pkg + all kris-* services (race + count=1)
test-all:
	@echo ">>> test pkg"
	@$(MAKE) -C pkg test || exit 1
	@for s in $(SERVICES); do \
	  echo ">>> test $$s"; \
	  $(MAKE) -C kris-$$s test || exit 1; \
	done

.PHONY: test-integration
# run pkg/data integration tests against local mongo/redis/mysql
test-integration:
	@$(MAKE) -C pkg test-integration

.PHONY: cover
# coverage profile for pkg + per-package summary
cover:
	@$(MAKE) -C pkg cover

.PHONY: cover-gate
# fail if total pkg coverage drops below COVER_MIN (default 75.0)
cover-gate:
	@$(MAKE) -C pkg cover-gate

# ---- DB migrations (golang-migrate) ----
#
# Usage:
#   make migrate-create SVC=alpha NAME=add_users
#   make migrate-up     SVC=alpha DB_URL='mysql://root:@tcp(127.0.0.1:3306)/kris'
#   make migrate-down   SVC=alpha DB_URL=... STEPS=1
#   make migrate-status SVC=alpha DB_URL=...
#
# Each service owns kris-<svc>/migrations/. Files are .sql, numbered by
# golang-migrate convention (NNNN_name.up.sql + NNNN_name.down.sql).

SVC      ?= alpha
DB_URL   ?=
STEPS    ?= 1
NAME     ?=
MIGR_DIR  = kris-$(SVC)/migrations

.PHONY: migrate-create
migrate-create:
ifndef NAME
	$(error usage: make migrate-create SVC=alpha NAME=add_users)
endif
	@mkdir -p $(MIGR_DIR)
	migrate create -ext sql -dir $(MIGR_DIR) -seq $(NAME)

.PHONY: migrate-up
migrate-up:
ifndef DB_URL
	$(error usage: make migrate-up SVC=alpha DB_URL='mysql://...')
endif
	migrate -database "$(DB_URL)" -path $(MIGR_DIR) up

.PHONY: migrate-down
migrate-down:
ifndef DB_URL
	$(error usage: make migrate-down SVC=alpha DB_URL='mysql://...' STEPS=1)
endif
	migrate -database "$(DB_URL)" -path $(MIGR_DIR) down $(STEPS)

.PHONY: migrate-status
migrate-status:
ifndef DB_URL
	$(error usage: make migrate-status SVC=alpha DB_URL='mysql://...')
endif
	migrate -database "$(DB_URL)" -path $(MIGR_DIR) version

.PHONY: vuln-check
# Go official vulnerability scan (call-graph aware) against every module.
# Install once: go install golang.org/x/vuln/cmd/govulncheck@latest
vuln-check:
	@echo ">>> vuln-check pkg"
	@cd pkg && govulncheck ./...
	@for s in $(SERVICES); do \
	  echo ">>> vuln-check kris-$$s"; \
	  (cd kris-$$s && govulncheck ./...) || exit 1; \
	done

.PHONY: bench
# run benchmarks across pkg
bench:
	@$(MAKE) -C pkg bench

DEMO_DIR := .demo
# Demo ports (override if these collide: make demo DEMO_ALPHA_GRPC=...)
DEMO_ALPHA_GRPC  ?= 28051
DEMO_ALPHA_HTTP  ?= 28080
DEMO_ALPHA_OTHER ?= 28081
DEMO_BETA_HTTP   ?= 28082
DEMO_BETA_OTHER  ?= 28083
DEMO_GAMMA_GRPC  ?= 28053
DEMO_GAMMA_OTHER ?= 28085

.PHONY: demo
# boot kris-alpha + kris-beta + kris-gamma in background; print curl commands.
# Default ports are in the 28xxx range to avoid common collisions.
demo: build-all
	@mkdir -p $(DEMO_DIR)
	@./kris-alpha/bin/alpha -grpc=:$(DEMO_ALPHA_GRPC) -http=:$(DEMO_ALPHA_HTTP) -other=:$(DEMO_ALPHA_OTHER) >$(DEMO_DIR)/alpha.log 2>&1 & echo $$! > $(DEMO_DIR)/alpha.pid
	@./kris-beta/bin/beta  -http=:$(DEMO_BETA_HTTP) -other=:$(DEMO_BETA_OTHER) >$(DEMO_DIR)/beta.log  2>&1 & echo $$! > $(DEMO_DIR)/beta.pid
	@./kris-gamma/bin/gamma -grpc=:$(DEMO_GAMMA_GRPC) -other=:$(DEMO_GAMMA_OTHER) -upstream=127.0.0.1:$(DEMO_ALPHA_GRPC) >$(DEMO_DIR)/gamma.log 2>&1 & echo $$! > $(DEMO_DIR)/gamma.pid
	@sleep 1
	@echo ""
	@echo "Booted. Logs under $(DEMO_DIR)/*.log; PIDs in $(DEMO_DIR)/*.pid."
	@echo ""
	@echo "Try:"
	@echo "  curl localhost:$(DEMO_ALPHA_HTTP)/                                              # alpha biz"
	@echo "  curl localhost:$(DEMO_ALPHA_OTHER)/version                                       # alpha sidecar"
	@echo "  curl localhost:$(DEMO_BETA_HTTP)/                                              # beta public"
	@echo "  curl -H 'Authorization: Bearer demo-alice' localhost:$(DEMO_BETA_HTTP)/whoami   # beta authed"
	@echo "  curl -X OPTIONS -H 'Origin: https://x' -H 'Access-Control-Request-Method: GET' \\"
	@echo "       -i localhost:$(DEMO_BETA_HTTP)/                                            # beta CORS preflight"
	@echo "  curl localhost:$(DEMO_GAMMA_OTHER)/readyz                                       # gamma probe"
	@echo ""
	@echo "Stop:  make demo-stop"

.PHONY: demo-stop
# kill the demo processes; tolerate missing pid files
demo-stop:
	@for svc in alpha beta gamma; do \
	  if [ -f $(DEMO_DIR)/$$svc.pid ]; then \
	    pid=$$(cat $(DEMO_DIR)/$$svc.pid); \
	    kill $$pid 2>/dev/null || true; \
	    rm $(DEMO_DIR)/$$svc.pid; \
	    echo "stopped $$svc ($$pid)"; \
	  fi; \
	done

.PHONY: ci-local
# mirror the GitHub CI matrix locally before pushing: build + vet + test + lint + fmt-check
ci-local:
	@echo ">>> build"
	@$(MAKE) build-all >/dev/null
	@echo ">>> vet"
	@$(MAKE) vet
	@echo ">>> test"
	@$(MAKE) test-all
	@echo ">>> lint"
	@$(MAKE) lint
	@echo ">>> fmt-check"
	@$(MAKE) fmt-check
	@echo "ci-local: all green"

.PHONY: gen-all
# proto + wire + tidy across all kris-* services
gen-all:
	@for s in $(SERVICES); do \
	  echo ">>> gen $$s"; \
	  $(MAKE) -C kris-$$s all || exit 1; \
	  $(MAKE) -C kris-$$s wire || exit 1; \
	done

# ---- Repo-wide code quality ----
.PHONY: lint
# golangci-lint across every module in the workspace
lint:
	golangci-lint run --build-tags=integration ./pkg/... $(SERVICES:%=./kris-%/...)

.PHONY: fmt
# gofumpt -extra + goimports across the whole repo (matches .golangci.yml rules)
fmt:
	gofumpt -extra -w pkg/ kris-*/
	goimports -w -local github.com/kris/go-infrastructure pkg/ kris-*/

.PHONY: lint-fix
# alias for fmt — auto-fix what golangci-lint would flag (gofumpt + goimports)
lint-fix: fmt

.PHONY: fmt-check
# verify-only: list files that gofumpt or goimports would change, fail if any.
# CI uses this to gate un-formatted PRs.
fmt-check:
	@out=$$(gofumpt -extra -l pkg/ kris-*/ 2>&1); \
	if [ -n "$$out" ]; then echo "gofumpt -extra would change:" && echo "$$out" && exit 1; fi
	@out=$$(goimports -l -local github.com/kris/go-infrastructure pkg/ kris-*/ 2>&1); \
	if [ -n "$$out" ]; then echo "goimports would change:" && echo "$$out" && exit 1; fi
	@echo "fmt-check: clean"

.PHONY: vet
# go vet across every module
vet:
	go vet ./pkg/... $(SERVICES:%=./kris-%/...)

# ---- Toolchain ----
.PHONY: tools-install
# install the pinned codegen toolchain (replaces per-service `make init`)
tools-install:
	bash tools/install.sh

# ---- Service scaffolding ----
.PHONY: new-service
# scaffold a new service; usage: make new-service NAME=foo GRPC=51000 HTTP=8200 OTHER=9200
new-service:
ifndef NAME
	$(error usage: make new-service NAME=<name> GRPC=<port> HTTP=<port> OTHER=<port>)
endif
	bash scripts/new-service.sh $(NAME) $(GRPC) $(HTTP) $(OTHER)

# ---- Local deps ----
.PHONY: dev-deps-up
# docker-compose up local mongo/redis/mysql/prometheus/grafana
dev-deps-up:
	docker compose -f docker-compose.dev.yml up -d

.PHONY: dev-deps-down
# stop local deps (volumes preserved)
dev-deps-down:
	docker compose -f docker-compose.dev.yml down

.PHONY: dev-deps-clean
# stop and remove volumes
dev-deps-clean:
	docker compose -f docker-compose.dev.yml down -v

# ---- Help ----
.PHONY: help
help:
	@echo 'go-infrastructure root Makefile -- orchestrating $(words $(SERVICES)) services'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "  \033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
