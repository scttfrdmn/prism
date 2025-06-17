.PHONY: build test clean install cross-compile

# Build for current platform
build:
	go build -o cws main.go

# Run tests
test:
	go test ./...

# Clean build artifacts
clean:
	rm -f cws cws-* *.exe

# Install locally
install: build
	cp cws /usr/local/bin/

# Cross-compile for all platforms
cross-compile: clean
	GOOS=linux GOARCH=amd64 go build -o cws-linux-amd64 main.go
	GOOS=linux GOARCH=arm64 go build -o cws-linux-arm64 main.go
	GOOS=darwin GOARCH=amd64 go build -o cws-darwin-amd64 main.go
	GOOS=darwin GOARCH=arm64 go build -o cws-darwin-arm64 main.go
	GOOS=windows GOARCH=amd64 go build -o cws-windows-amd64.exe main.go

# Quick test with actual AWS (requires AWS credentials)
test-launch:
	./cws templates
	@echo "To test launch: ./cws launch basic-ubuntu test-instance"

# Development mode - build and show usage
dev: build
	./cws

# Package for release
package: cross-compile
	mkdir -p dist
	mv cws-* dist/
	cd dist && tar -czf cws-linux-amd64.tar.gz cws-linux-amd64
	cd dist && tar -czf cws-linux-arm64.tar.gz cws-linux-arm64
	cd dist && tar -czf cws-darwin-amd64.tar.gz cws-darwin-amd64
	cd dist && tar -czf cws-darwin-arm64.tar.gz cws-darwin-arm64
	cd dist && zip cws-windows-amd64.zip cws-windows-amd64.exe
