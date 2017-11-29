.PHONY: install all-image image-%

# The binary to build (just the basename).
BIN := einbuchen
#ALL_ARCH := amd64 arm arm64 ppc64le
ALL_ARCH ?= amd64

# This repo's root import path (under GOPATH).
# PKG ?= $(shell realpath --relative-to=${GOPATH}/src `pwd`)
PKG ?= github.com/container-bootcamp-demo/einbuchen

VERSION ?= "latest"

all-image: $(addprefix image-, $(ALL_ARCH))

image-%: ARCH = $*
image-%: Dockerfile-%
	docker build -t quay.io/containerbootcamp/$(BIN):$(VERSION) -f Dockerfile-$(ARCH) .
	docker push quay.io/containerbootcamp/$(BIN):$(VERSION)

Dockerfile-%: ARCH = $*
Dockerfile-%: Dockerfile.in
	@sed \
		-e 's|ARG_BIN|$(BIN)|g' \
		-e 's|ARG_PKG|$(PKG)|g' \
		-e 's|ARG_ARCH|$(ARCH)|g' \
		Dockerfile.in > $@

## dev stuff
Gopkg.toml:
	dep init

vendor: Gopkg.toml
	dep ensure

install: vendor
	go install ./...