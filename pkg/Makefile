.PHONY: test
# unit test the pkg module
test:
	go test -race -count=1 ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: tidy
tidy:
	go mod tidy

.DEFAULT_GOAL := test
