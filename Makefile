.PHONY: all godmesg test

all: godmesg

godmesg:
	go build -o ./bin/godmesg ./cmd/godmesg

test:
	go test -v ./...
