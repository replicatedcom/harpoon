# Harpoon
Harpoon is a go library and command line tool to pull a Docker image from any source and load it into Docker.  
This works by manually downloading the manifest and each layer blob and metadata, then assembling this into
a .tar.gz file that is compatible with a `docker load` command.  Then, harpoon will actually load the image
into Docker.


## Usage
harpoon pull <flags> <image_uri>

Possible flags:
`--proxy <value>` Use this http(s) proxy server when pulling the image
`--no-cache` Ignore the docker cache, if available.
`--no-load` Download and leave the image as a .tar.gz file without loading it into Docker.
`--force-v1` Force use of the v1 registry protocol.
`--username` The username to authenticate to the registry with.
`--password` The password to authenticate to the registry with.
`--token` Use the supplied token to pull the image.  (Not compatible with registry protocol v1 or v2 (only v2.2))

Image URI should be in the format of:
`docker://<server>/<namespace>/<image>:<tag>`

Examples:
Pull the public, official nginx container:
docker://nginx

Pull a private image from docker hub named "priv", tag "abc", owned by docker hub organization "org":
docker://org/priv:abc

Pull a private image from quay.io named "priv", tag "abc", owned by quay.io organization "org":
docker://quay.io/org/priv:abc


## Contributing
```shell
docker build -t harpoon .
make shell
```

### Testing

Some tests require credentials to interact with Docker hub.  No data will be changed, but you should
supply the required data in the environment before running the tests.
```shell
export DOCKERHUB_USERNAME=<a valid dockerhub username>
export DOCKERHUB_PASSWORD=<a valid dockerhub password>
make test
```
