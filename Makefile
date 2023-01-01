SHELL=/usr/bin/env bash

all: ffi build

unexport GOFLAGS

ldflags=-X=github.com/llifezou/fil-wallet/build.CurrentCommit=+git.$(subst -,.,$(shell git describe --always --match=NeVeRmAtCh 2>/dev/null || git rev-parse --short HEAD 2>/dev/null))
ifneq ($(strip $(LDFLAGS)),)
 ldflags+=-extldflags=$(LDFLAGS)
endif

GOFLAGS+=-ldflags="$(ldflags)"

.PHONY: ffi
ffi:
	git submodule update --init --recursive
	./extern/filecoin-ffi/install-filcrypto

.PHONY: build
build: build
	go mod tidy
	rm -rf fil-wallet
	go build $(GOFLAGS) -o fil-wallet ./main.go

