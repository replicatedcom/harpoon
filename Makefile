export GO111MODULE=on

.PHONY: test
test:
	go test --race -v ./...

.PHONY: clean
clean:
	rm -f ./bin/harpoon

.PHONY: build
build:
	mkdir -p ./bin
	go build -o ./bin/harpoon .
