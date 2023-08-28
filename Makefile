.PHONY: all build run clean

# Default target
all: build

# Build the project
build:
	go build -o hotreload

# Run the built binary
run:
	go run main.go builder.go

# Clean build artifacts
clean:
	rm -f hotreload
