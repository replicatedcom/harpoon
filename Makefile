.PHONY: all test clean deps build runr

test:
	@for pkg in $(shell cat ./test/packages.txt); do \
		godep go test -covermode=count $${pkg}; \
	done

clean:
	rm -rf ./Godeps/_workspace/pkg
	rm -rf ./_vendor
	rm -f ./bin/harpoon

deps:
	godep save ./...

build:
	mkdir -p ./bin
	godep go build -o ./bin/harpoon .

shell:
	docker run --rm -it -P --name harpoon \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/github.com/replicatedhq/harpoon \
		-e DOCKERHUB_USERNAME=$(DOCKERHUB_USERNAME) \
		-e DOCKERHUB_PASSWORD=$(DOCKERHUB_PASSWORD) \
		harpoon
