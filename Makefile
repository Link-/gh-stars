# Makefile with the following targets:
#   all: build the project
#   clean: remove all build artifacts
#   test: run the tests
#   run: run the project
#   build: build the project
#   build-docker: build the project in a docker container
#   run-docker: run the project in a docker container
#   test-docker: run the tests in a docker container
#   clean-docker: remove all build artifacts in a docker container
#   help: print this help message
#   .PHONY: mark targets as phony
#   .DEFAULT_GOAL: set the default goal to all

# Set the default goal to all
.DEFAULT_GOAL := all
PROJECT_NAME := "gh-stars"

# Mark targets as phony
.PHONY: all clean test run build build-docker run-docker test-docker clean-docker help

# Build the project
all: clean build test

# Remove all build artifacts
clean:
	rm gh-stars

# Run the tests
test: build
	go test ./...

# Run the tests and print a rich output
test-rich: build
	go test -v ./...

# Build the project
build:
	go build -o gh-stars .
