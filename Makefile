.PHONY: build test clean

build:
	go build -o bin/nrs ./cmd/nrs

test:
	go test ./... -v

clean:
	rm -rf bin/
