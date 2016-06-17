.PHONY: all test clean deps build shell vendor

test:
	# go get github.com/stretchr/testify/assert
	govendor test -v +local

clean:
	rm -rf ./go
	rm -f ./bin/harpoon

vendor:
	# initial setup
	# to add new repos, run "govendor fetch <url>"
	go get -t ./...
	govendor init
	govendor add +external

build:
	mkdir -p ./bin
	govendor build -o ./bin/harpoon .

shell:
	docker run --rm -it -P --name harpoon \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/github.com/replicatedhq/harpoon \
		-v /tmp:/tmp \
		-e DOCKERHUB_USERNAME=$(DOCKERHUB_USERNAME) \
		-e DOCKERHUB_PASSWORD=$(DOCKERHUB_PASSWORD) \
		harpoon
