
GOBIN:=$(GOBIN)

all: test example

test:
	$(GOBIN) test -race ./...
	$(GOBIN) test -tags no_json ./...
	$(GOBIN) test -tags no_http ./...

local-coverage: coverage
	$(GOBIN) test -coverprofile=coverage.out ./...
	$(GOBIN) tool cover -html=coverage.out

coverage:
	$(GOBIN) test -coverprofile=coverage1.out ./...
	$(GOBIN) test -tags no_net -coverprofile=coverage2.out ./...
	$(GOBIN) test -tags no_json -coverprofile=coverage3.out ./...
	$(GOBIN) test -tags no_http,no_json -coverprofile=coverage4.out ./...
	# cat coverage*.out > coverage.out
	$(GOBIN) install github.com/wadey/gocovmerge@b5bfa59ec0adc420475f97f89b58045c721d761c
	gocovmerge coverage?.out > coverage.out

example:
	@echo "### Colorized (default) ###"
	$(GOBIN) run ./levelsDemo
	@echo "### JSON: (redirected stderr) ###"
	$(GOBIN) run ./levelsDemo 3>&1 1>&2 2>&3 | jq -c

line:
	@echo

# Suitable to make a screenshot with a bit of spaces around for updating color.png
screenshot: line example
	@echo

size-check:
	@echo "### Size of the binary:"
	CGO_ENABLED=0 $(GOBIN) build -ldflags="-w -s" -trimpath -o ./fullsize ./levelsDemo
	ls -lh ./fullsize
	CGO_ENABLED=0 $(GOBIN) build -tags no_net -ldflags="-w -s" -trimpath -o ./smallsize ./levelsDemo
	ls -lh ./smallsize
	CGO_ENABLED=0 $(GOBIN) build -tags no_http,no_json -ldflags="-w -s" -trimpath -o ./smallsize ./levelsDemo
	ls -lh ./smallsize
	gsa ./smallsize # $(GOBIN) install github.com/Zxilly/$(GOBIN)-size-analyzer/cmd/gsa@master


lint: .golangci.yml
	golangci-lint run
	golangci-lint run --build-tags no_json

.golangci.yml: Makefile
	curl -fsS -o .golangci.yml https://raw.githubusercontent.com/fortio/workflows/main/golangci.yml


.PHONY: all test example screenshot line lint
