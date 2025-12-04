.PHONY: all build test clean docker-build

# Variables
APP_NAME=webhook
DOCKER_IMAGE=harvester-nested-virt-webhook
VERSION?=latest
GOOS?=linux
GOARCH?=amd64

all: test build

# Build the binary
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(APP_NAME) ./cmd/webhook

# Run tests
test:
	go test -v -race -coverprofile=coverage.out ./...

# Run tests with coverage
test-coverage: test
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Run linter (requires golangci-lint)
lint:
	golangci-lint run ./...

# Format code
fmt:
	go fmt ./...

# Tidy dependencies
tidy:
	go mod tidy

# Build Docker image
docker-build:
	docker build -t $(DOCKER_IMAGE):$(VERSION) .

# Run the webhook locally (requires certificates)
run: build
	./bin/$(APP_NAME)

# Install dependencies
deps:
	go mod download
