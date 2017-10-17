package remote

import (
	"os"
	"testing"

	"github.com/replicatedcom/harpoon/log"

	"github.com/stretchr/testify/require"
)

func TestAuth(t *testing.T) {
	dockerRemote := DockerRemote{
		Hostname:       os.Getenv("REGISTRY_HOSTNAME"),
		Token:          os.Getenv("REGISTRY_TOKEN"),
		Username:       os.Getenv("REGISTRY_USERNAME"),
		Password:       os.Getenv("REGISTRY_PASSWORD"),
		PreferredProto: "v2",
	}

	var err error

	err = dockerRemote.InitClient()
	require.NoError(t, err)
	err = dockerRemote.Auth()
	require.NoError(t, err)
	log.Debugf("remote info:%#v", dockerRemote)
}
