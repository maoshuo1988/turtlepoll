# Build the application
all: build

build:
	@echo "Building..."
	@go build -v -o bbs-go main.go

buildlinux:
	@echo "Building..."
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o bbs-go-linux main.go

# Run the application
run:
	@go run main.go

# Test the application
test:
	@echo "Testing..."
	@go test ./...

# Clean the binary
clean:
	@echo "Cleaning..."
	@rm -f bbs-go

generator:
	@go run cmd/generator/generator.go

.PHONY: all build run test clean
