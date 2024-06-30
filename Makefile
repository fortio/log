
all: test example

test:
	go test -race ./...
	go test -tags no_json ./...
	go test -tags no_http ./...

example:
	@echo "### Colorized (default) ###"
	go run ./levelsDemo
	@echo "### JSON: (redirected stderr) ###"
	go run ./levelsDemo 3>&1 1>&2 2>&3 | jq -c

line:
	@echo

# Suitable to make a screenshot with a bit of spaces around for updating color.png
screenshot: line example
	@echo

size-check:
	@echo "### Size of the binary:"
	CGO_ENABLED=0 go build -ldflags="-w -s" -trimpath -o ./fullsize ./levelsDemo
	ls -lh ./fullsize
	CGO_ENABLED=0 go build -tags no_net -ldflags="-w -s" -trimpath -o ./smallsize ./levelsDemo
	ls -lh ./smallsize
	CGO_ENABLED=0 go build -tags no_http,no_json -ldflags="-w -s" -trimpath -o ./smallsize ./levelsDemo
	ls -lh ./smallsize
	gsa ./smallsize # go install github.com/Zxilly/go-size-analyzer/cmd/gsa@master


lint: .golangci.yml
	golangci-lint run

.golangci.yml: Makefile
	curl -fsS -o .golangci.yml https://raw.githubusercontent.com/fortio/workflows/main/golangci.yml


.PHONY: all test example screenshot line lint
