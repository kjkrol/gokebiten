GO = go

.PHONY: all demo-collision deps tidy test clean

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

clean:
	$(GO) clean
