.PHONY: test lint fmt

GO ?= go

fmt:
	$(GO) fmt ./...

test:
	$(GO) test ./...

lint:
	golangci-lint run ./...
