.PHONY: build test fetch run run-all report clean

BENCH_BIN := bin/bench

build:
	CGO_ENABLED=1 go build -o $(BENCH_BIN) ./cmd/bench

test:
	go test ./...

fetch:
	$(BENCH_BIN) fetch-tasks --source swe-bench-lite --count 25 --output ./tasks/sets/swe-bench-sm

run:
	$(BENCH_BIN) run --set swe-bench-sm --model claude-sonnet-4-6

run-all:
	$(BENCH_BIN) run --set swe-bench-sm --model claude-sonnet-4-6 --scenario all

report:
	@if [ -z "$(DIR)" ]; then echo "Usage: make report DIR=./results/<timestamp>"; exit 1; fi
	$(BENCH_BIN) report --dir $(DIR)

clean:
	rm -rf bin/ results/
