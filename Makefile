.PHONY: clean build check test

clean:
	rm -f ./slack-rage

build: clean
	go build

check:
	go fmt ./...
	go vet ./...

test:
	go test -cover -v ./...
