.PHONY: all build test clean

all: build

build:
	go build -o mira cmd/miracula/main.go

test:
	go test ./...

clean:
	rm -f mira
