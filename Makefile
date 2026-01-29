VERSION := dev-$(shell date +%Y-%m-%d_%H:%M:%S)
INSTALL_DIR := ~/pink-tools/pink-transcriber

build:
	go build -ldflags="-X main.version=$(VERSION)" -o pink-transcriber ./cmd/pink-transcriber

install: build
	cp pink-transcriber $(INSTALL_DIR)/pink-transcriber
