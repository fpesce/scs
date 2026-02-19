export PATH := /usr/local/go/bin:/home/joke/go/bin:$(PATH)

.PHONY: lint test

lint:
	golangci-lint run ./...

test:
	CGO_ENABLED=1 go test -v -race ./...

