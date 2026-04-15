.PHONY: build test clean

build:
	go build -o bin/mss ./cmd/mss

test:
	go test ./... -v

clean:
	rm -rf bin/
