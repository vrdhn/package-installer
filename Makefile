VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "unknown")
TIMESTAMP := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X pi/pkg/config.BuildVersion=$(VERSION) -X pi/pkg/config.BuildTimestamp=$(TIMESTAMP)

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o pi main.go

.PHONY: install
install:
	go install -ldflags "$(LDFLAGS)"

.PHONY: test
test:
	go test ./...

.PHONY: clean
clean:
	rm -f pi
