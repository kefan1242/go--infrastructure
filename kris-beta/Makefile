VERSION   := $(shell git describe --tags --always 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse HEAD 2>/dev/null || echo "unknown")
BUILDTIME := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

NAME         := beta
SERVICE_NAME := kris-$(NAME)

LDFLAGS := -X main.Name=$(SERVICE_NAME) \
           -X main.Version=$(VERSION) \
           -X main.Commit=$(COMMIT) \
           -X main.BuildTime=$(BUILDTIME)

.PHONY: build
build:
	mkdir -p bin/
	go build -ldflags "$(LDFLAGS)" -o ./bin/$(NAME) ./cmd/$(NAME)

.PHONY: run
run: build
	./bin/$(NAME)

.PHONY: test
test:
	go test -race -count=1 ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: tidy
tidy:
	GOWORK=off go mod tidy

.PHONY: all
all: tidy

.DEFAULT_GOAL := build
