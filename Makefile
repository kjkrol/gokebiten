GO = go

.PHONY: all demo run deps tidy test clean

all: demo

## demo: Alias for run — fetches dependencies and launches the collision-demo example
demo: run

demo-nav: run-board-nav

demo-scenes: run-scenes

## run: Fetches dependencies and launches the collision-demo example
run: deps
	$(GO) run ./examples/collision-demo

run-board-nav: deps
	$(GO) run ./examples/board-navigation-demo

run-scenes: deps
	$(GO) run ./examples/scenes-demo

deps:
	$(GO) mod tidy

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...

clean:
	$(GO) clean
