GO ?= go
DIST ?= dist
.PHONY: build test test-race lint integration benchmark package clean fmt verify
build:
	@mkdir -p $(DIST)
	$(GO) build -o $(DIST)/sentineld ./cmd/sentineld
	$(GO) build -o $(DIST)/sentinel ./cmd/sentinel
test:
	$(GO) test ./... -count=1
test-race:
	$(GO) test -race ./...
lint:
	@test -z "$$($(GO)fmt -l .)"
	$(GO) vet ./...
	@if command -v staticcheck >/dev/null 2>&1; then staticcheck ./...; fi
integration:
	$(GO) test ./test/integration -count=1
benchmark:
	$(GO) test -bench=. -run='^$$' ./...
package:
	bash ./packaging/debian/build.sh
clean:
	$(GO) clean -cache -testcache
	rm -rf $(DIST) packaging/debian/out packaging/debian/stage
fmt:
	$(GO)fmt -w $$(find . -name '*.go' -not -path './vendor/*')
verify: lint test test-race integration build
