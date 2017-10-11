package proxy

import (
	"os"
	"strings"
	"testing"

	"github.com/replicatedcom/harpoon/log"
	"github.com/replicatedcom/harpoon/remote"

	"github.com/stretchr/testify/require"
)

func TestAuth(t *testing.T) {
	dockerRemote := &remote.DockerRemote{
		Hostname:       os.Getenv("REGISTRY_HOSTNAME"),
		Token:          os.Getenv("REGISTRY_TOKEN"),
		Username:       os.Getenv("REGISTRY_USERNAME"),
		Password:       os.Getenv("REGISTRY_PASSWORD"),
		PreferredProto: "v2",
	}

	var err error

	log.Debugf("calling InitClient")
	err = dockerRemote.InitClient()
	require.NoError(t, err)

	// hack parsing
	parts := strings.Split(os.Getenv("PRIVATE_IMAGE"), "/") // like, quay.io/replicatedcom/market-api:973f05b
	imageParts := strings.Split(parts[2], ":")

	p := &Proxy{Remote: dockerRemote}
	manifest, err := p.GetManifestV2(parts[1], imageParts[0], imageParts[1])
	require.NoError(t, err)
	log.Debugf("manifest:%#v", manifest)
}
