GOLANGCI_LINT_VERSION := v2.15.0
GOBIN := $(shell go env GOPATH)/bin

.PHONY: dev build clean lint install-lint

dev:
	wails3 dev

build:
	wails3 build

clean:
	rm -rf bin frontend/dist .task Whip

lint:
	$(GOBIN)/golangci-lint run ./...

install-lint:
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(GOBIN) $(GOLANGCI_LINT_VERSION)
