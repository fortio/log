
all: test example

test:
	go test . -race ./...

example:
	@echo "### Colorized (default) ###"
	go run ./levelsDemo
	@echo "### JSON: (redirected stderr) ###"
	go run ./levelsDemo 3>&1 1>&2 2>&3 | jq -c

.PHONY: all test example
