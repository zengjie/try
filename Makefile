.PHONY: build build-all test clean install

BINARY_NAME=try
MAIN_PATH=main.go
BUILD_DIR=dist

build:
	go build -o $(BINARY_NAME) $(MAIN_PATH)

build-all:
	mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	GOOS=linux GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(MAIN_PATH)
	GOOS=darwin GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	GOOS=windows GOARCH=amd64 go build -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)
	rm -rf $(BUILD_DIR)
	go clean

install: build
	mkdir -p ~/.local/bin
	cp $(BINARY_NAME) ~/.local/bin/
	@echo "Try has been installed to ~/.local/bin/$(BINARY_NAME)"
	@echo "Make sure ~/.local/bin is in your PATH"
	@echo ""
	@echo "To complete the installation, add this to your shell config:"
	@echo "  eval \"\$$(try init ~/src/tries)\""

run: build
	./$(BINARY_NAME)

deps:
	go mod download
	go mod tidy

fmt:
	go fmt ./...

vet:
	go vet ./...

lint: fmt vet
	@echo "Code has been formatted and vetted"