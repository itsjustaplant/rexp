.PHONY: build check test

build:
	go build -o bin/rexp .

check:
	go vet ./...
	@test -z "$$(gofmt -l main.go internal)" || { gofmt -l main.go internal; exit 1; }

test:
	go test ./...
