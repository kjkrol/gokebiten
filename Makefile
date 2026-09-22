GO = go

COMMIT_HASH := $(shell git rev-parse --short HEAD)
COMMIT_DATE := $(shell git log -1 --format=%cd --date=format:%Y%m%d)
DIRTY       := $(shell git diff --quiet || echo "-dirty")
RESULT_FILE := bench_results/bench_$(COMMIT_DATE)_$(COMMIT_HASH)$(DIRTY).txt
BENCH_COUNT ?= 5

.PHONY: all demo-collision demo-navigation demo-scenes demo-vision deps tidy test bench bench-save clean

all: demo-collision

## demo: Alias for run — fetches dependencies and launches the collision-demo example
demo-collision: run-collision

demo-navigation: run-navigation

demo-scenes: run-scenes

demo-vision: run-vision

## run: Fetches dependencies and launches the collision-demo example
run-collision: deps
	$(GO) run ./examples/collision-demo

run-navigation: deps
	$(GO) run ./examples/navigation-demo

run-scenes: deps
	$(GO) run ./examples/scenes-demo

run-vision: deps
	$(GO) run ./examples/vision-demo

deps:
	$(GO) mod tidy

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...

## bench: runs the whole suite once, with allocations
bench:
	$(GO) test -bench=. -benchmem -count=1 ./bench/...

## bench-save: runs the suite BENCH_COUNT times and keeps the raw output under bench_results/
bench-save:
	@mkdir -p bench_results
	$(GO) test -bench=. -benchmem -count=$(BENCH_COUNT) ./bench/... > $(RESULT_FILE)
	@echo "Results saved into: $(RESULT_FILE)"

clean:
	$(GO) clean
