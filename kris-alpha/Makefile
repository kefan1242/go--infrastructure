# kris-alpha service Makefile.
#
# This is also the template scripts/new-service.sh clones from.
# Add proto / wire / openapi targets here when your service grows past the
# minimal scaffold.

GOHOSTOS := $(shell go env GOHOSTOS)
GOPATH   := $(shell go env GOPATH)
VERSION  := $(shell git describe --tags --always 2>/dev/null || echo "dev")
COMMIT   := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILDTIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

NAME         := alpha
SERVICE_NAME := kris-$(NAME)

LDFLAGS := -X main.Name=$(SERVICE_NAME) \
           -X main.Version=$(VERSION) \
           -X main.Commit=$(COMMIT) \
           -X main.BuildTime=$(BUILDTIME)

# Shared proto includes (cross-service)
THIRD_PARTY := ../pkg/third_party

.PHONY: build
# build the service binary with Name/Version/Commit/BuildTime injected
build:
	mkdir -p bin/
	go build -ldflags "$(LDFLAGS)" -o ./bin/$(NAME) ./cmd/$(NAME)

IMAGE_REPO ?= kris/$(NAME)
IMAGE_TAG  ?= $(VERSION)

.PHONY: docker
# build the docker image; runs from the repo root so the pkg/ replace target is in context
docker:
	cd .. && docker build \
	  -f kris-$(NAME)/Dockerfile \
	  --build-arg VERSION=$(VERSION) \
	  --build-arg COMMIT=$(COMMIT) \
	  --build-arg BUILDTIME=$(BUILDTIME) \
	  -t $(IMAGE_REPO):$(IMAGE_TAG) .

.PHONY: run
# build + run with default flags
run: build
	./bin/$(NAME)

.PHONY: test
# unit test (race + count=1)
test:
	go test -race -count=1 ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: tidy
# `go mod tidy` with workspace mode off so we get the module-local view
tidy:
	GOWORK=off go mod tidy

.PHONY: generate
generate:
	GOWORK=off go generate ./...

.PHONY: wire
# regenerate wire_gen.go (no-op until you add a wire.go)
wire:
	@if [ -d cmd/$(NAME) ] && ls cmd/$(NAME)/wire.go >/dev/null 2>&1; then \
	  cd cmd/$(NAME) && wire; \
	else \
	  echo "no cmd/$(NAME)/wire.go — skipping"; \
	fi

# ---- Codegen (only fires when api/ exists) ----

ifeq ($(GOHOSTOS),windows)
	Git_Bash := $(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
	INTERNAL_PROTO_FILES := $(shell $(Git_Bash) -c "find internal -name '*.proto' 2>/dev/null")
	API_PROTO_FILES      := $(shell $(Git_Bash) -c "find api -name '*.proto' 2>/dev/null")
else
	INTERNAL_PROTO_FILES := $(shell find internal -name '*.proto' 2>/dev/null)
	API_PROTO_FILES      := $(shell find api -name '*.proto' 2>/dev/null)
endif

.PHONY: config
# generate conf.pb.go from internal/conf/*.proto
config:
	@if [ -n "$(INTERNAL_PROTO_FILES)" ]; then \
	  protoc --proto_path=./internal \
	         --proto_path=$(THIRD_PARTY) \
	         --go_out=paths=source_relative:./internal \
	         $(INTERNAL_PROTO_FILES); \
	else \
	  echo "no internal/*.proto — skipping"; \
	fi

.PHONY: api
# generate go / go-grpc / go-http / openapi from api/**/*.proto
api:
	@if [ -n "$(API_PROTO_FILES)" ]; then \
	  protoc --proto_path=./api \
	         --proto_path=$(THIRD_PARTY) \
	         --go_out=paths=source_relative:./api \
	         --go-http_out=paths=source_relative:./api \
	         --go-grpc_out=paths=source_relative:./api \
	         --openapi_out=fq_schema_naming=true,default_response=false:. \
	         $(API_PROTO_FILES); \
	else \
	  echo "no api/*.proto — skipping"; \
	fi

.PHONY: errors
errors:
	@if [ -n "$(API_PROTO_FILES)" ]; then \
	  protoc --proto_path=./api \
	         --proto_path=$(THIRD_PARTY) \
	         --go_out=paths=source_relative:./api \
	         --go-errors_out=paths=source_relative:./api \
	         $(API_PROTO_FILES); \
	else \
	  echo "no api/*.proto — skipping"; \
	fi

.PHONY: validate
validate:
	@if [ -n "$(API_PROTO_FILES)" ]; then \
	  protoc --proto_path=./api \
	         --proto_path=$(THIRD_PARTY) \
	         --go_out=paths=source_relative:./api \
	         --validate_out=paths=source_relative,lang=go:./api \
	         $(API_PROTO_FILES); \
	else \
	  echo "no api/*.proto — skipping"; \
	fi

.PHONY: all
# regenerate everything: api + config + tidy
all: api config tidy

.DEFAULT_GOAL := build
