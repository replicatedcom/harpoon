package dockerreg

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLibraryNoTag(t *testing.T) {
	dockerRemote, err := ParseDockerURI("docker://img")
	require.NoError(t, err)

	assert.Equal(t, DefaultHostname, dockerRemote.Hostname)
	assert.Equal(t, DefaultNamespace, dockerRemote.Namespace)
	assert.Equal(t, "img", dockerRemote.ImageName)
	assert.Equal(t, DefaultTag, dockerRemote.Tag)
}

func TestParseLibraryTag(t *testing.T) {
	dockerRemote, err := ParseDockerURI("docker://img:tag")
	require.NoError(t, err)

	assert.Equal(t, DefaultHostname, dockerRemote.Hostname)
	assert.Equal(t, DefaultNamespace, dockerRemote.Namespace)
	assert.Equal(t, "img", dockerRemote.ImageName)
	assert.Equal(t, "tag", dockerRemote.Tag)
}

func TestParseNamespaceNoTag(t *testing.T) {
	dockerRemote, err := ParseDockerURI("docker://ns/img")
	require.NoError(t, err)

	assert.Equal(t, DefaultHostname, dockerRemote.Hostname)
	assert.Equal(t, "ns", dockerRemote.Namespace)
	assert.Equal(t, "img", dockerRemote.ImageName)
	assert.Equal(t, DefaultTag, dockerRemote.Tag)
}

func TestParseNamespaceTag(t *testing.T) {
	dockerRemote, err := ParseDockerURI("docker://ns/img:tag")
	require.NoError(t, err)

	assert.Equal(t, DefaultHostname, dockerRemote.Hostname)
	assert.Equal(t, "ns", dockerRemote.Namespace)
	assert.Equal(t, "img", dockerRemote.ImageName)
	assert.Equal(t, "tag", dockerRemote.Tag)
}

func TestParseNotHubNoTag(t *testing.T) {
	dockerRemote, err := ParseDockerURI("docker://hostname.com/ns/img")
	require.NoError(t, err)

	assert.Equal(t, "hostname.com", dockerRemote.Hostname)
	assert.Equal(t, "ns", dockerRemote.Namespace)
	assert.Equal(t, "img", dockerRemote.ImageName)
	assert.Equal(t, DefaultTag, dockerRemote.Tag)
}

func TestParseNotHubTag(t *testing.T) {
	dockerRemote, err := ParseDockerURI("docker://hostname.com/ns/img:tag")
	require.NoError(t, err)

	assert.Equal(t, "hostname.com", dockerRemote.Hostname)
	assert.Equal(t, "ns", dockerRemote.Namespace)
	assert.Equal(t, "img", dockerRemote.ImageName)
	assert.Equal(t, "tag", dockerRemote.Tag)
}
