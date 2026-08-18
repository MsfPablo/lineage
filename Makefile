.PHONY: test build run

test:
	go test ./...

build:
	go build -o bin/lineage ./cmd/lineage

run:
	go run ./cmd/lineage --help
