package proxy

import (
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
	"github.com/replicatedcom/harpoon/remote"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPull(t *testing.T) {
	dockerRemote := &remote.DockerRemote{
		Hostname:       os.Getenv("REGISTRY_HOSTNAME"),
		Token:          os.Getenv("REGISTRY_TOKEN"),
		Username:       os.Getenv("REGISTRY_USERNAME"),
		Password:       os.Getenv("REGISTRY_PASSWORD"),
		PreferredProto: "v2",
	}

	named, err := reference.ParseNormalizedNamed(os.Getenv("PRIVATE_IMAGE"))
	require.NoError(t, err)
	dockerRemote.Ref = reference.TagNameOnly(named)

	log.Println("calling InitClient")
	err = dockerRemote.InitClient()
	require.NoError(t, err)

	// hack parsing
	parts := strings.Split(os.Getenv("PRIVATE_IMAGE"), "/") // like, quay.io/replicatedcom/market-api:973f05b
	imageParts := strings.Split(parts[2], ":")

	p := &Proxy{
		Remote: dockerRemote,
	}

	manifestResult, err := p.GetManifestV2(parts[1], imageParts[0], imageParts[1], []string{schema2.MediaTypeManifest})
	require.NoError(t, err)
	log.Printf("manifest JSON:\n%s", manifestResult.SignedJson)

	type Layer struct {
		BlobSum string `json:"blobSum"`
	}
	type Manifest struct {
		FSLayers []Layer `json:"fsLayers"`
	}

	manifest := &Manifest{}
	err = json.Unmarshal(manifestResult.SignedJson, manifest)
	require.NoError(t, err)
	log.Printf("layers:\n%#v", manifest)

	assert.NotEmpty(t, manifest.FSLayers)

	// this will download 2 layers...
	for i := 1; i < 3; i++ {
		blobResult, err := p.GetBlobV2(parts[1], imageParts[0], manifest.FSLayers[i].BlobSum, nil)
		require.NoError(t, err)

		log.Printf("blobResult:\n%#v", blobResult)
		defer blobResult.Reader.Close()
		n, err := io.Copy(ioutil.Discard, blobResult.Reader)
		require.NoError(t, err)

		log.Printf("copied %d bytes", n)
		assert.Equal(t, blobResult.ContentLength, n)
	}
}

func Test_GetBlobV2(t *testing.T) {
	// Private image hosted in Replicated QA project
	// gcr.io/replicated-qa/qa-ubuntu@sha256:bc025862c3e8ec4a8754ea4756e33da6c41cba38330d7e324abd25c8e0b93300
	p := &Proxy{
		Remote: &remote.DockerRemote{
			Hostname:       "gcr.io",
			Token:          "",
			Username:       "",
			Password:       "",
			PreferredProto: "v2",
		},
	}
	err := p.Remote.InitClient()
	assert.NoError(t, err)

	resp, err := p.GetBlobV2("replicated-qa", "qa-ubuntu", "sha256:bc025862c3e8ec4a8754ea4756e33da6c41cba38330d7e324abd25c8e0b93300", nil)
	assert.Error(t, err)
	assert.Nil(t, resp)

	proxyError, ok := errors.Cause(err).(*ProxyError)
	assert.True(t, ok)
	assert.Equal(t, http.StatusUnauthorized, proxyError.StatusCode)
}
