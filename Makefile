GOBIN ?= $(shell go env GOPATH)/bin

.PHONY: all
all: test

.PHONY: test
test:
	go test -v -race ./...

.PHONY: lint
lint: $(GOBIN)/golint
	go vet ./...
	golint -set_exit_status ./...

$(GOBIN)/golint:
	cd && go get golang.org/x/lint/golint

.PHONY: clean
clean:
	go clean
