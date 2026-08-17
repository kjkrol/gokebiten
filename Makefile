GO = go

.PHONY: all demo run deps tidy test clean

all: demo

## demo: Alias for run — fetches dependencies and launches the collision-demo example
demo: run

## run: Fetches dependencies and launches the collision-demo example
run: deps
	$(GO) run ./examples/collision-demo

deps:
	$(GO) mod tidy

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...

clean:
	$(GO) clean
