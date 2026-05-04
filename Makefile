GO ?= go
DIST ?= dist
VERSION ?= dev
GOOS ?= $(shell $(GO) env GOOS)
GOARCH ?= $(shell $(GO) env GOARCH)

.PHONY: build test package clean

build:
	$(GO) build -o bin/mss ./cmd/mss

test:
	$(GO) test ./... -v

package: build
	mkdir -p $(DIST)
	LC_ALL=C tar -C bin -czf $(DIST)/mss-$(VERSION)-$(GOOS)-$(GOARCH).tar.gz mss

clean:
	rm -rf bin/ $(DIST)/
