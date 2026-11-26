GOLANGCI_LINT_VERSION := v2.15.0
GOBIN := $(shell go env GOPATH)/bin

# App version (semver, e.g. 0.1.2-alpha). The default lives in Go
# (version.Default in internal/app/version/version.go); set it for a build with e.g.
#   make distribute VERSION=0.2.0-beta
# Make exports command-line variables, so the Wails build (build/darwin/Taskfile.yml)
# sees VERSION and stamps it into the binary with -ldflags -X.

# Version checker; prints MAJOR.MINOR.PATCH (or the full version with -full).
SEMVER := go run ./cmd/semver

.PHONY: dev build dmg distribute version sync-version clean lint install-lint

dev:
	wails3 dev

build:
	wails3 build

# Print the version a build would use.
version:
	@$(SEMVER) -full $(VERSION)

# Validate VERSION and copy its MAJOR.MINOR.PATCH into build/config.yml, which
# drives Info.plist (Apple allows only plain x.y.z there) and the Windows/Linux
# package metadata. Fails before anything is built if VERSION isn't valid semver.
sync-version:
	@core=$$($(SEMVER) $(VERSION)) || exit 1; \
	sed -i '' -E "s/^(  version: )\"[^\"]*\"/\\1\"$$core\"/" build/config.yml; \
	echo "Version $$($(SEMVER) -full $(VERSION)) (Info.plist: $$core)"

dmg: sync-version
	wails3 task common:update:build-assets
	wails3 task darwin:package:universal
	wails3 task darwin:create:dmg DMG_VOLUME_ICON=build/darwin/gecko.icns DMG_FILE_ICON=build/darwin/gecko.icns
	# The universal binary is now inside Whip.app; remove the loose copy so later
	# native builds (make dev / make build) can write bin/Whip again.
	rm -f bin/Whip

# Release build: requires an explicit version, e.g. make distribute VERSION=0.2.0-beta
distribute:
	@test -n "$(VERSION)" || { echo "make distribute needs a version, e.g. make distribute VERSION=0.2.0-beta" >&2; exit 1; }
	$(MAKE) dmg
	cp bin/Whip.dmg ~/Desktop/

clean:
	rm -rf bin frontend/dist .task Whip

lint:
	$(GOBIN)/golangci-lint run ./...

install-lint:
	curl -sSfL https://golangci-lint.run/install.sh | sh -s -- -b $(GOBIN) $(GOLANGCI_LINT_VERSION)
