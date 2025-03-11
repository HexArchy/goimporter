.PHONY: build test install clean install-global install-gobin format lint

# Tool name
BINARY_NAME=goimporter

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOLINT=golangci-lint

# Build flags
BUILD_FLAGS=-v
LDFLAGS=-s -w

all: test build

build:
	$(GOBUILD) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

test:
	go test ./...

install:
	$(GOINSTALL) -ldflags "$(LDFLAGS)" .

clean:
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

lint:
	$(GOLINT) run

# Format and organize imports of the project itself
format: build
	./$(BINARY_NAME) -r

# System-wide installation (macOS/Linux only)
install-global: build
	sudo cp $(BINARY_NAME) /usr/local/bin/

# Install to Go bin directory (preferred for personal use)
install-gobin:
	$(GOBUILD) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o ~/go/bin/$(BINARY_NAME) .