.PHONY: build run clean test

VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

build:
	go build -ldflags="-s -w -X main.version=$(VERSION)" -o kbflash ./cmd/kbflash

run: build
	./kbflash

clean:
	rm -f kbflash

test:
	go test ./...
