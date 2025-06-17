#!/bin/bash
# Development helper script

case "$1" in
    "build")
        echo "🔨 Building..."
        go build -o cws main.go
        echo "✅ Built: ./cws"
        ;;
    "test")
        echo "🧪 Testing..."
        go test ./...
        ;;
    "run")
        echo "🏃 Building and running..."
        go build -o cws main.go && ./cws "${@:2}"
        ;;
    "clean")
        echo "🧹 Cleaning..."
        rm -f cws cws-* *.exe
        ;;
    *)
        echo "Usage: ./dev.sh [build|test|run|clean]"
        echo "  build: Build the application"
        echo "  test:  Run tests"
        echo "  run:   Build and run with args"
        echo "  clean: Clean build artifacts"
        ;;
esac
