TAG ?= latest
GO_FILES := $(shell find . -type f -name '*.go')

.PHONY: build
build: pdf-converter

pdf-converter: $(GO_FILES)
	go build -o $@ .

.PHONY: image
image: build
	docker build --no-cache -t ghcr.io/ymmt2005/pdf-converter:$(TAG) .
