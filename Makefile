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
	golangci-lint run ./pkg/... ./kris-*/...

.PHONY: fmt
# gofumpt + goimports across the whole repo
fmt:
	gofumpt -w pkg/ kris-*/
	goimports -w -local github.com/kris/go-infrastructure pkg/ kris-*/

.PHONY: vet
# go vet across every module
vet:
	go vet ./pkg/... ./kris-*/...

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
