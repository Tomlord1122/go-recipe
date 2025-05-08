# Binary name
BINARY_NAME=go-recipe

# Build directory
BUILD_DIR=build

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod

# Main entry point
MAIN_PKG=./cmd/go-recipe

# Version information from git
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Linker flags
LDFLAGS=-ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(BUILD_DATE)"

.PHONY: all build clean test tidy run

all: clean build

build:
	mkdir -p $(BUILD_DIR)
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(LDFLAGS) $(MAIN_PKG)

clean:
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

test:
	$(GOTEST) -v ./...

tidy:
	$(GOMOD) tidy

run:
	$(GOBUILD) -o $(BUILD_DIR)/$(BINARY_NAME) $(LDFLAGS) $(MAIN_PKG)
	./$(BUILD_DIR)/$(BINARY_NAME)

install:
	$(GOBUILD) -o $(GOPATH)/bin/$(BINARY_NAME) $(LDFLAGS) $(MAIN_PKG) 