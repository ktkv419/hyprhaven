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
	GOOS=linux GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

build-windows-debug:
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME).exe .

build-windows:
	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS) -H windowsgui" -o $(BINARY_NAME).exe .

build-windows-dev:
	$(shell taskkill.exe /F /IM hyprhaven.exe > /dev/null 2>&1 || true)	

	GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME).exe .
	
	cp $(BINARY_NAME).exe /mnt/c/Users/$(shell cmd.exe /c echo %USERNAME% | tr -d '\r')/


build-darwin:
	GOOS=darwin GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o $(BINARY_NAME) .

