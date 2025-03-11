.PHONY: build test install clean install-global install-gobin format lint install-vk

# Tool name
BINARY_NAME=goimporter

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOINSTALL=$(GOCMD) install
GOCLEAN=$(GOCMD) clean
GOLINT=golangci-lint

# Build flags
BUILD_FLAGS=-v
LDFLAGS=-s -w

all: test build

build:
	$(GOBUILD) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) ./goimporter/cmd/goimporter

test:
	$(GOTEST) -v ./...

install:
	$(GOINSTALL) -ldflags "$(LDFLAGS)" ./goimporter/cmd/goimporter

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
	$(GOBUILD) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" -o ~/go/bin/$(BINARY_NAME) ./goimporter/cmd/goimporter

# Install with VK-specific configuration
install-vk:
	./install_vk.sh