BINARY_NAME=upp
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS=-ldflags "-X github.com/naru-bot/upp/cmd.Version=$(VERSION)"

.PHONY: build clean test install cross

build:
	CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME) .

install:
	CGO_ENABLED=0 go install $(LDFLAGS) .

test:
	go test ./...

clean:
	rm -f $(BINARY_NAME) upp-*

# Cross-compile for all platforms
cross:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .
