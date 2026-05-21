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

.PHONY: cover
# unit-test coverage profile + total line
cover:
	go test -race -count=1 -coverprofile=coverage.out ./...
	@echo ""
	@go tool cover -func=coverage.out | tail -1

.PHONY: cover-html
# open the coverage profile as an HTML report
cover-html: cover
	go tool cover -html=coverage.out -o coverage.html
	@echo "open: $(PWD)/coverage.html"

.PHONY: bench
# run benchmarks across the module
bench:
	go test -run='^$$' -bench=. -benchmem ./...

.DEFAULT_GOAL := test
