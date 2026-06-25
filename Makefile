GO ?= go
BIN_DIR ?= bin
BINARY ?= sing-box-sub
CMD ?= ./cmd/sing-box-subscribe-cli
BIN := $(BIN_DIR)/$(BINARY)

.PHONY: all build test clean list list-template

all: build

build:
	mkdir -p $(BIN_DIR)
	$(GO) build -o $(BIN) $(CMD)

test:
	$(GO) test ./...

clean:
	rm -f $(BIN)

list:
	$(GO) run $(CMD) list

list-template: list
