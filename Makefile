TAG ?= latest
GO_FILES := $(shell find . -type f -name '*.go')

.PHONY: build
build: pdf-converter

.PHONY: test
test: test-tools
	go test -race -v -count 1 ./...
	go install ./sub ./converter
	go vet ./...
	test -z $$(gofmt -s -l . | tee /dev/stderr)
	staticcheck ./...

pdf-converter: $(GO_FILES)
	go build -o $@ .

.PHONY: image
image: build
	docker build --no-cache -t ghcr.io/ymmt2005/pdf-converter:$(TAG) .
	docker tag ghcr.io/ymmt2005/pdf-converter:${TAG} quay.io/ymmt2005/pdf-converter:${TAG} 

.PHONY: test-tools
test-tools: staticcheck

.PHONY: staticcheck
staticcheck:
	if ! which staticcheck >/dev/null; then \
		cd /tmp; env GOFLAGS= GO111MODULE=on go get honnef.co/go/tools/cmd/staticcheck; \
	fi
