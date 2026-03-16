#!/usr/bin/env bash

set -e

# Diskimager Build Script

COMMAND=$1

case "$COMMAND" in
    build)
        echo "Building diskimager..."
        go build -o diskimager .
        echo "Build complete."
        ;;
    clean)
        echo "Cleaning build artifacts..."
        rm -f diskimager
        rm -f *.log *.img *.e01 *.bin *.dd
        rm -rf recovered_files_*/
        echo "Clean complete."
        ;;
    test)
        echo "Running unit tests..."
        go test -v ./...
        ;;
    tidy)
        echo "Tidying go modules..."
        go mod tidy
        ;;
    *)
        echo "Usage: ./build.sh {build|clean|test|tidy}"
        exit 1
        ;;
esac
