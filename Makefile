.PHONY: build run test lint tidy

build:
	go build -o bin/server ./cmd/server

run:
	go run ./cmd/server

test:
	go test ./... -count=1

lint:
	go vet ./...

tidy:
	go mod tidy
