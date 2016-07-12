package dockerreg

import (
	"archive/tar"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/replicatedhq/harpoon/log"

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
	dockerRemote.PreferredProto = "v2"
	readCloser, err := dockerRemote.StreamLayers()
	require.NotNil(t, readCloser)
	defer readCloser.Close()

	tarReader := tar.NewReader(readCloser)
	for {
		hdr, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		require.NoError(t, err)
		log.Debugf("Extracting %d bytes for file %s", hdr.Size, hdr.Name)

		_, err = io.CopyN(ioutil.Discard, tarReader, hdr.Size)
		require.NoError(t, err)
	}
}
