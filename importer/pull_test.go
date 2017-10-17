package importer

import (
	"os"
	"testing"

	"github.com/replicatedcom/harpoon/remote"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSupportedProtocol(t *testing.T) {
	dockerRemote, err := remote.ParseDockerURI("docker://redis:3.0.5")
	require.NoError(t, err)

	i := Importer{Remote: dockerRemote}

	var ok bool

	i.Remote.PreferredProto = "v1"
	ok, err = i.isSupportedProtocol()
	require.NoError(t, err)
	assert.True(t, ok)

	i.Remote.PreferredProto = "v2"
	ok, err = i.isSupportedProtocol()
	require.NoError(t, err)
	assert.True(t, ok)
}

func TestPullImagePrivate(t *testing.T) {
	image := os.Getenv("PRIVATE_IMAGE")
	token := os.Getenv("REGISTRY_TOKEN")
	username := os.Getenv("REGISTRY_USERNAME")
	password := os.Getenv("REGISTRY_PASSWORD")

	dockerRemote, err := remote.ParseDockerURI("docker://" + image)
	require.NoError(t, err)

	dockerRemote.Token = token
	dockerRemote.Username = username
	dockerRemote.Password = password
	dockerRemote.PreferredProto = "v2"
	imageImporter := Importer{
		Remote: dockerRemote,
	}
	readCloser, err := imageImporter.StreamLayers()
	require.NotNil(t, readCloser)
	defer readCloser.Close()

	err = imageImporter.ImportFromStream(readCloser, image)
	require.NoError(t, err)
}
