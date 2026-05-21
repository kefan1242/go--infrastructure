VERSION := $(shell git describe --tags --always 2>/dev/null || echo "dev")
NAME    := gamma

.PHONY: build
build:
	mkdir -p bin/
	go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/$(NAME) ./cmd/$(NAME)

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
