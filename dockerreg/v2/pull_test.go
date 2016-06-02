package v2

import (
	"os"
	"testing"

	"github.com/replicatedhq/harpoon/dockerreg"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSupportedProtocol(t *testing.T) {
	dockerRemote := dockerreg.DockerRemote{
		Hostname:  "index.docker.io",
		Namespace: "library",
		ImageName: "redis",
		Tag:       "3.0.5",

		Username: os.Getenv("DOCKERHUB_USERNAME"),
		Password: os.Getenv("DOCKERHUB_PASSWORD"),
	}

	ok, err := isSupportedProtocol(&dockerRemote)
	require.NoError(t, err)

	assert.True(t, ok)
}

func TestPullImage(t *testing.T) {
	dockerRemote := dockerreg.DockerRemote{
		Hostname:  "index.docker.io",
		Namespace: "library",
		ImageName: "redis",
		Tag:       "3.0.5",

		Username: os.Getenv("DOCKERHUB_USERNAME"),
		Password: os.Getenv("DOCKERHUB_PASSWORD"),
	}

	err := PullImage(&dockerRemote, "", false, "")
	require.NoError(t, err)
}
