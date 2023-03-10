gofiles = $(shell find . -type f -name \*.go)

bin = bin/bla

default: fmt test $(bin)

$(bin): $(gofiles)
	go build -o $(bin) *.go 

fmt: $(gofiles)
	go fmt ./...

test: $(gofiles)
	go test ./...

