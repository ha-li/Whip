GOLANGCI_LINT_VERSION := v2.15.0
GOBIN := $(shell go env GOPATH)/bin

.PHONY: dev build dmg distribute clean lint install-lint

dev:
	wails3 dev

build:
	wails3 build

dmg:
	wails3 task common:update:build-assets
	wails3 task darwin:package:universal
	wails3 task darwin:create:dmg DMG_VOLUME_ICON=build/darwin/gecko.icns DMG_FILE_ICON=build/darwin/gecko.icns
	# The universal binary is now inside Whip.app; remove the loose copy so later
	# native builds (make dev / make build) can write bin/Whip again.
	rm -f bin/Whip

distribute: dmg
	cp bin/Whip.dmg ~/Desktop/

clean:
	rm -rf bin frontend/dist .task Whip

lint:
	$(GOBIN)/golangci-lint run ./...

install-lint:
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(GOBIN) $(GOLANGCI_LINT_VERSION)
