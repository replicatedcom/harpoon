.PHONY: all test clean install build docker shell

test:
	go test --race `go list ./... | grep -v /vendor/`

clean:
	rm -f ./bin/harpoon

install:
	govendor install +local +vendor,^program

build:
	mkdir -p ./bin
	go build -o ./bin/harpoon .

docker:
	docker build -t harpoon .

shell:
	docker run --rm -it -P --name harpoon \
		--add-host registry.replicated.com:192.168.100.100 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src/github.com/replicatedcom/harpoon \
		-v /tmp:/tmp \
		-e PRIVATE_IMAGE=$(PRIVATE_IMAGE) \
		-e REGISTRY_TOKEN=$(REGISTRY_TOKEN) \
		-e REGISTRY_USERNAME=$(REGISTRY_USERNAME) \
		-e REGISTRY_PASSWORD=$(REGISTRY_PASSWORD) \
		-e REGISTRY_HOSTNAME=$(REGISTRY_HOSTNAME) \
		harpoon
