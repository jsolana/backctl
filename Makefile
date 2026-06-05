VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    ?= $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)"
DOCKER_IMAGE ?= backctl-mcp
DOCKER_TAG   ?= $(VERSION)

.PHONY: build build-mcp test lint install clean docker-build

build:
	go build $(LDFLAGS) -o bin/backctl ./cmd/backctl

build-mcp:
	go build $(LDFLAGS) -o bin/backctl-mcp ./cmd/backctl-mcp

docker-build:
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg COMMIT=$(COMMIT) \
		--build-arg DATE=$(DATE) \
		-t $(DOCKER_IMAGE):$(DOCKER_TAG) \
		-t $(DOCKER_IMAGE):latest \
		.

test:
	go test -race ./...

TESTABLE = ./internal/client ./internal/entityref ./internal/output ./internal/resolver

coverage:
	go test -coverprofile=coverage.out -covermode=atomic $(TESTABLE)
	go tool cover -func=coverage.out

coverage-html:
	go test -coverprofile=coverage.out -covermode=atomic $(TESTABLE)
	go tool cover -html=coverage.out -o coverage.html
	open coverage.html

lint:
	golangci-lint run

install:
	go install $(LDFLAGS) ./cmd/backctl

clean:
	rm -rf bin/
