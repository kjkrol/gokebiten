GO = go

.PHONY: all demo run deps tidy test clean

all: demo

## demo: Alias for run — fetches dependencies and launches the collision-demo example
demo: run

demo-rts: run-board

## run: Fetches dependencies and launches the collision-demo example
run: deps
	$(GO) run ./examples/collision-demo

run-board: deps
	$(GO) run ./examples/board-demo

deps:
	$(GO) mod tidy

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...

clean:
	$(GO) clean
