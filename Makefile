GOARCH ?= $(shell go env GOARCH)
BUILD_ARGS := -trimpath -ldflags "-s -w" $(BUILD_ARGS)
OUTPUT ?= feedbot
GOOS ?= $(shell go env GOOS)

.PHONY: feedbot

ifndef CGO_ENABLED
feedbot: export CGO_ENABLED=0
endif
feedbot:
	go build -o $(OUTPUT) $(BUILD_ARGS) cmd/bot/main.go