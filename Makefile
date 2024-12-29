# Define variables for easy configuration
GO=go
BINARY_NAME=hyprhaven
LDFLAGS=-s -w

# Default target, will build the binary
all: build

# Build the binary for the current platform
build:
	$(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

# Run tests
test:
	$(GO) test ./...

# Clean up build artifacts
clean:
	rm -f $(BINARY_NAME)*

# Run the application
run:
	./$(BINARY_NAME)

# Format Go code
fmt:
	$(GO) fmt ./...

# Install the binary to GOPATH/bin (if necessary)
install:
	$(GO) install .

# Generate Go documentation (optional)
doc:
	$(GO) doc

# Cross-compile for different platforms (Linux/Windows/macOS)
build-linux:
	GOOS=linux GOARCH=amd64 $(GO) build -o $(BINARY_NAME) .

build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build -o $(BINARY_NAME).exe .

build-darwin:
	GOOS=darwin GOARCH=amd64 $(GO) build -o $(BINARY_NAME) .

