SHELL=/usr/bin/env bash

all: ffi build

.PHONY: ffi
ffi:
	git submodule update --init --recursive
	./extern/filecoin-ffi/install-filcrypto

.PHONY: build
build: build
	go mod tidy
	rm -rf fil-wallet
	go build -o fil-wallet ./main.go

