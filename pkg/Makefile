.PHONY: test
# unit test the pkg module
test:
	go test -race -count=1 ./...

.PHONY: test-integration
# integration tests against local mongo/redis/mysql (build tag: integration)
# override endpoints via KRIS_MONGO_URI / KRIS_REDIS_ADDR / KRIS_MYSQL_DSN.
test-integration:
	go test -tags integration -race -count=1 ./data/...

.PHONY: vet
vet:
	go vet ./...

.PHONY: tidy
tidy:
	go mod tidy

.DEFAULT_GOAL := test
