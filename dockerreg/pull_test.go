package dockerreg

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSupportedProtocol(t *testing.T) {
	dockerRemote, err := ParseDockerURI("docker://redis:3.0.5")
	require.NoError(t, err)

	var ok bool

	dockerRemote.PreferredProto = "v1"
	ok, err = isSupportedProtocol(dockerRemote)
	require.NoError(t, err)
	assert.True(t, ok)

	dockerRemote.PreferredProto = "v2"
	ok, err = isSupportedProtocol(dockerRemote)
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestPullImagePrivate(t *testing.T) {
	image := os.Getenv("PRIVATE_IMAGE")
	token := os.Getenv("REGISTRY_TOKEN")
	username := os.Getenv("REGISTRY_USERNAME")
	password := os.Getenv("REGISTRY_PASSWORD")

	dockerRemote, err := ParseDockerURI("docker://" + image)
	require.NoError(t, err)

	dockerRemote.Token = token
	dockerRemote.Username = username
	dockerRemote.Password = password

	//  dockerRemote.PreferredProto = "v1"
	//	store1, err := dockerRemote.PullImage(false)
	//  if assert.NotNil(t, store1) {
	//    require.NoError(t, store1.delete())
	//  }
	//	require.NoError(t, err)

	dockerRemote.PreferredProto = "v2"
	store2, err := dockerRemote.PullImage(false)
	if assert.NotNil(t, store2) {
		require.NoError(t, store2.delete())
	}
	require.NoError(t, err)
}
