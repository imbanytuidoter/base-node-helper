.PHONY: build test test-cover lint vet fmt clean

GO ?= go
BIN := bin/base-node-helper
PKGS := ./...

build:
	$(GO) build -o $(BIN) ./cmd/base-node-helper

test:
	$(GO) test $(PKGS)

test-cover:
	$(GO) test -coverprofile=coverage.txt -covermode=atomic $(PKGS)
	$(GO) tool cover -func=coverage.txt | tail -1

vet:
	$(GO) vet $(PKGS)

fmt:
	$(GO) fmt $(PKGS)

lint:
	$(GO) run honnef.co/go/tools/cmd/staticcheck@latest $(PKGS)

clean:
	rm -rf bin dist coverage.txt
