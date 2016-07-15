package dockerreg

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/replicatedcom/harpoon/log"
	"github.com/replicatedcom/harpoon/requests"

	"github.com/docker/distribution/digest"
	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/docker/image"
	"github.com/docker/docker/image/v1"
	"github.com/docker/docker/layer"
	"github.com/docker/docker/reference"
)

func (dockerRemote *DockerRemote) StreamLayers() (io.ReadCloser, error) {
	pipeReader, pipeWriter := io.Pipe()
	go dockerRemote.writeLayers(pipeWriter)
	return pipeReader, nil
}

func (dockerRemote *DockerRemote) writeLayers(pipeWriter *io.PipeWriter) {
	tarWriter := tar.NewWriter(pipeWriter)

	var writeError error
	defer func() {
		pipeWriter.CloseWithError(writeError)
		tarWriter.Close()
	}()

	supported, writeError := isSupportedProtocol(dockerRemote)
	if writeError != nil {
		return
	}
	if !supported {
		writeError = errors.New("Docker registry v2 protocol is not supported by remote")
		return
	}

	dockerRemote.JWTToken = ""
	rawManifest, writeError := dockerRemote.getManifestBytes()
	if writeError != nil {
		return
	}

	tarHeader := &tar.Header{
		Name: ManifestFileNameV2,
		// Mode: 0655,
		Size: int64(len(rawManifest)),
	}
	if writeError = tarWriter.WriteHeader(tarHeader); writeError != nil {
		log.Error(writeError)
		return
	}

	if _, writeError = tarWriter.Write(rawManifest); writeError != nil {
		log.Error(writeError)
		return
	}

	var manifest schema1.SignedManifest
	if writeError = manifest.UnmarshalJSON(rawManifest); writeError != nil {
		log.Error(writeError)
		return
	}

	// Send layers in reverse because import needs to read them in reverse order
	for i := len(manifest.FSLayers) - 1; i >= 0; i-- {
		layer := manifest.FSLayers[i]

		var blobStream io.ReadCloser
		var expectedLenght int64

		blobStream, expectedLenght, writeError = dockerRemote.getBlobStream(layer.BlobSum)
		if writeError != nil {
			return
		}
		defer blobStream.Close() // ok to keep open until func terminates

		tarHeader := &tar.Header{
			Name: layer.BlobSum.String(),
			Size: expectedLenght,
		}
		if writeError = tarWriter.WriteHeader(tarHeader); writeError != nil {
			log.Error(writeError)
			return
		}

		_, writeError = io.Copy(tarWriter, blobStream)
		if writeError != nil {
			log.Error(writeError)
			return
		}
	}
}

// ImportFromStream will read manifest and layer data from a single tar stream
func ImportFromStream(reader io.Reader, imageURI string) error {
	tmpStore, err := streamToTempStore(reader, imageURI)
	if tmpStore != nil {
		defer tmpStore.delete()
	}
	if err != nil {
		return err
	}
	return ImportFromLocal(tmpStore)
}

func streamToTempStore(reader io.Reader, imageURI string) (*v1Store, error) {
	ref, err := reference.ParseNamed(imageURI)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	tarReader := tar.NewReader(reader)
	verifiedManifest, err := getManifestFromTar(tarReader, ref)
	if err != nil {
		return nil, err
	}

	localStore, err := getV1Store(verifiedManifest)
	if err != nil {
		return nil, err
	}

	rootFS := image.NewRootFS()
	var history []image.History
	var parent digest.Digest
	layerV1IDs := make([]digest.Digest, 0)

	for i := len(verifiedManifest.FSLayers) - 1; i >= 0; i-- {
		layer := verifiedManifest.FSLayers[i]

		var throwAway struct {
			ThrowAway bool `json:"throwaway,omitempty"`
		}

		v1ImageJson := []byte(verifiedManifest.History[i].V1Compatibility)
		if err := json.Unmarshal(v1ImageJson, &throwAway); err != nil {
			return localStore, err
		}

		h, err := v1.HistoryFromConfig(v1ImageJson, throwAway.ThrowAway)
		if err != nil {
			return localStore, err
		}
		history = append(history, h)

		if throwAway.ThrowAway {
			log.Debugf("Skipping throw away layer: %s", layer.BlobSum.String())
			if err := skipLayerInTar(tarReader, layer.BlobSum); err != nil {
				return localStore, err
			}
			continue
		}

		// Create layer.tar, json, and VERSION files for the layer in a temp folder because v1 layer ID is not known ahead of time.
		layerTempDir, err := ioutil.TempDir(localStore.Workspace, "tmp_layer")
		if err != nil {
			log.Error(err)
			return localStore, err
		}

		blobSum := layer.BlobSum
		diffID, err := downloadBlobFromTar(tarReader, blobSum, layerTempDir)
		if err != nil {
			return localStore, err
		}

		if err = ioutil.WriteFile(filepath.Join(layerTempDir, "VERSION"), []byte("1.0"), 0644); err != nil {
			log.Error(err)
			return localStore, err
		}

		rootFS.Append(diffID) // rootFS must contain this layer ID to produce correct chain ID
		v1Img := image.V1Image{}
		if i == 0 {
			if err := json.Unmarshal(v1ImageJson, &v1Img); err != nil {
				log.Error(err)
				return localStore, err
			}
		}

		v1ID, err := v1.CreateID(v1Img, rootFS.ChainID(), parent)
		if err != nil {
			log.Error(err)
			return localStore, err
		}

		v1Img.ID = v1ID.Hex()
		if len(parent.String()) > 0 {
			v1Img.Parent = parent.Hex()
		}
		v1ImgJson, err := json.Marshal(v1Img)
		if err != nil {
			log.Error(err)
			return localStore, err
		}
		if err := ioutil.WriteFile(filepath.Join(layerTempDir, "json"), v1ImgJson, 0644); err != nil {
			log.Error(err)
			return localStore, err
		}

		layerDir := filepath.Join(localStore.Workspace, v1ID.Hex())

		log.Debugf("Moving %s to %s", layerTempDir, layerDir)
		if err := os.Rename(layerTempDir, layerDir); err != nil {
			log.Error(err)
			return localStore, err
		}

		layerV1IDs = append(layerV1IDs, v1ID)
		parent = v1ID
	}

	config, err := v1.MakeConfigFromV1Config([]byte(verifiedManifest.History[0].V1Compatibility), rootFS, history)
	if err != nil {
		return localStore, err
	}

	imageID := image.ID(digest.FromBytes(config))

	if err := localStore.writeConfigFile(imageID, config); err != nil {
		return localStore, err
	}

	if err := localStore.writeRepositoriesFile(ref, imageID); err != nil {
		return localStore, err
	}

	if err := localStore.writeManifestFile(ref, imageID, layerV1IDs); err != nil {
		return localStore, err
	}

	return localStore, nil
}

// PullImage will pull image from v2 with v1 (todo) fallback
func (dockerRemote *DockerRemote) PullImage() (*v1Store, error) {
	// Validate that the remote server supports the v2 protocol

	if dockerRemote.PreferredProto == "v1" {
		return dockerRemote.PullImageV1()
	}

	supported, err := isSupportedProtocol(dockerRemote)
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, errors.New("Docker registry v2 protocol is not supported by remote")
	}

	// Ugh, this isn't the right design to use here.
	// But the token will be set from checking the /v2/ endpoint without a scope, which will cause a
	// 401 when trying to pull.
	dockerRemote.JWTToken = ""

	verifiedManifest, err := dockerRemote.getManifest()
	if err != nil {
		return nil, err
	}

	if len(verifiedManifest.FSLayers) == 0 {
		err := fmt.Errorf("Can't export/import images without layers")
		log.Error(err)
		return nil, err
	}

	localStore, err := getV1Store(verifiedManifest)
	if err != nil {
		return nil, err
	}

	rootFS := image.NewRootFS()
	var history []image.History
	var parent digest.Digest
	layerV1IDs := make([]digest.Digest, 0)

	// Note that we check number of layers above so it's safe to run loop with i == 0
	for i := len(verifiedManifest.FSLayers) - 1; i >= 0; i-- {
		layer := verifiedManifest.FSLayers[i]

		var throwAway struct {
			ThrowAway bool `json:"throwaway,omitempty"`
		}

		v1ImageJson := []byte(verifiedManifest.History[i].V1Compatibility)
		if err := json.Unmarshal(v1ImageJson, &throwAway); err != nil {
			return localStore, err
		}

		h, err := v1.HistoryFromConfig(v1ImageJson, throwAway.ThrowAway)
		if err != nil {
			return localStore, err
		}
		history = append(history, h)

		if throwAway.ThrowAway {
			log.Debugf("Skipping throw away layer: %s", layer.BlobSum.String())
			continue
		}

		// Create layer.tar, json, and VERSION files for the layer in a temp folder because v1 layer ID is not known ahead of time.
		layerTempDir, err := ioutil.TempDir(localStore.Workspace, "tmp_layer")
		if err != nil {
			log.Error(err)
			return localStore, err
		}

		blobSum := layer.BlobSum
		diffID, err := downloadBlob(dockerRemote, blobSum, layerTempDir)
		if err != nil {
			return localStore, err
		}

		if err = ioutil.WriteFile(filepath.Join(layerTempDir, "VERSION"), []byte("1.0"), 0644); err != nil {
			log.Error(err)
			return localStore, err
		}

		rootFS.Append(diffID) // rootFS must contain this layer ID to produce correct chain ID
		v1Img := image.V1Image{}
		if i == 0 {
			if err := json.Unmarshal(v1ImageJson, &v1Img); err != nil {
				log.Error(err)
				return localStore, err
			}
		}

		v1ID, err := v1.CreateID(v1Img, rootFS.ChainID(), parent)
		if err != nil {
			log.Error(err)
			return localStore, err
		}

		v1Img.ID = v1ID.Hex()
		if len(parent.String()) > 0 {
			v1Img.Parent = parent.Hex()
		}
		v1ImgJson, err := json.Marshal(v1Img)
		if err != nil {
			log.Error(err)
			return localStore, err
		}
		if err := ioutil.WriteFile(filepath.Join(layerTempDir, "json"), v1ImgJson, 0644); err != nil {
			log.Error(err)
			return localStore, err
		}

		layerDir := filepath.Join(localStore.Workspace, v1ID.Hex())

		log.Debugf("Moving %s to %s", layerTempDir, layerDir)
		if err := os.Rename(layerTempDir, layerDir); err != nil {
			log.Error(err)
			return localStore, err
		}

		layerV1IDs = append(layerV1IDs, v1ID)
		parent = v1ID
	}

	config, err := v1.MakeConfigFromV1Config([]byte(verifiedManifest.History[0].V1Compatibility), rootFS, history)
	if err != nil {
		return localStore, err
	}

	imageID := image.ID(digest.FromBytes(config))

	if err := localStore.writeConfigFile(imageID, config); err != nil {
		return localStore, err
	}

	if err := localStore.writeRepositoriesFile(dockerRemote.Ref, imageID); err != nil {
		return localStore, err
	}

	if err := localStore.writeManifestFile(dockerRemote.Ref, imageID, layerV1IDs); err != nil {
		return localStore, err
	}

	return localStore, nil
}

// downloadBlob will download and write the layer to the workDir, in the docker format
func downloadBlob(dockerRemote *DockerRemote, blobsum digest.Digest, layerDir string) (layer.DiffID, error) {
	uri := fmt.Sprintf("https://%s/v2/%s/%s/blobs/%s", dockerRemote.Hostname, dockerRemote.Namespace, dockerRemote.ImageName, blobsum.String())

	fmt.Printf("Downloading blob from %q\n", uri)

	client := requests.GlobalHttpClient()

	req, err := client.NewRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return layer.DiffID(""), err
	}

	resp, err := doRequest(req, client, dockerRemote, 0)
	if err != nil {
		return layer.DiffID(""), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("Unexpected status code for %s: %d", uri, resp.StatusCode)
		log.Error(err)
		return layer.DiffID(""), err
	}

	gzipDigest := digest.Canonical.New()
	responseReader := io.TeeReader(resp.Body, gzipDigest.Hash())

	archive, err := gzip.NewReader(responseReader)
	if err != nil {
		log.Errorf("Failed to create gzip reader: %v", err)
		return layer.DiffID(""), err
	}
	defer archive.Close()

	target := filepath.Join(layerDir, "layer.tar")
	writer, err := os.Create(target)
	if err != nil {
		log.Errorf("Failed to create tar file %s: %v", target, err)
		return layer.DiffID(""), err
	}
	defer writer.Close()

	tarDigest := digest.Canonical.New()
	tarWriter := io.MultiWriter(writer, tarDigest.Hash())

	_, err = io.Copy(tarWriter, archive)
	if err != nil {
		log.Error(err)
		return layer.DiffID(""), err
	}

	computedBlobsum := digest.Digest(gzipDigest.Digest())
	if blobsum.String() != computedBlobsum.String() {
		err := fmt.Errorf("Downloaded layer blobsum does not match expected blobsum: %s != %s", blobsum, computedBlobsum)
		log.Error(err)
		return layer.DiffID(""), err
	}

	diffID := digest.Digest(tarDigest.Digest())
	log.Debugf("Downloaded layer %s, with blobsum %s", diffID, computedBlobsum)

	return layer.DiffID(diffID), nil
}

func (dockerRemote *DockerRemote) getBlobStream(blobsum digest.Digest) (io.ReadCloser, int64, error) {
	uri := fmt.Sprintf("https://%s/v2/%s/%s/blobs/%s", dockerRemote.Hostname, dockerRemote.Namespace, dockerRemote.ImageName, blobsum.String())

	fmt.Printf("Downloading blob from %q\n", uri)

	client := requests.GlobalHttpClient()

	req, err := client.NewRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return nil, 0, err
	}

	resp, err := doRequest(req, client, dockerRemote, 0)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		err := fmt.Errorf("Unexpected status code for %s: %d", uri, resp.StatusCode)
		log.Error(err)
		return nil, 0, err
	}

	fmt.Printf("Responded with content-length: %q\n", resp.Header.Get("Content-Length"))
	expectedSize, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		log.Error(err)
		expectedSize = -1
	}

	return resp.Body, expectedSize, nil
}

// getManifest will return the remote manifest for the image.
func (dockerRemote *DockerRemote) getManifest() (*schema1.Manifest, error) {
	rawManifest, err := dockerRemote.getManifestBytes()
	if err != nil {
		return nil, err
	}

	var manifest schema1.SignedManifest
	if err := manifest.UnmarshalJSON(rawManifest); err != nil {
		log.Error(err)
		return nil, err
	}

	fmt.Printf("manifest = %#v\n", manifest)

	verifiedManifest, err := verifySchema1Manifest(&manifest, dockerRemote.Ref)
	if err != nil {
		return nil, err
	}

	fmt.Printf("verifiedManifest = %#v\n", verifiedManifest)

	return verifiedManifest, nil
}

func (dockerRemote *DockerRemote) getManifestBytes() ([]byte, error) {
	uri := fmt.Sprintf("https://%s/v2/%s/%s/manifests/%s", dockerRemote.Hostname, dockerRemote.Namespace, dockerRemote.ImageName, dockerRemote.Tag)

	client := requests.GlobalHttpClient()

	req, err := client.NewRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v1+json")

	resp, err := doRequest(req, client, dockerRemote, 0)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("Unexpected status code for %s: %d", uri, resp.StatusCode)
		log.Error(err)
		return nil, err
	}

	contentType := resp.Header.Get("Content-Type")
	fmt.Printf("Responded with content-type: %q\n", contentType)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	return body, nil
}

func getManifestFromTar(tarReader *tar.Reader, ref reference.Named) (*schema1.Manifest, error) {
	hdr, err := tarReader.Next()
	if err != nil { // EOF is also an error here.  We need the manifest.
		err := fmt.Errorf("Cannot read manifest from tar stream: %v", err)
		log.Error(err)
		return nil, err
	}

	if hdr.Name != ManifestFileNameV2 {
		err := fmt.Errorf("Expected %q but found %q", ManifestFileNameV2, hdr.Name)
		log.Error(err)
		return nil, err
	}

	manifestBuffer := bytes.NewBuffer(nil)
	_, err = io.CopyN(manifestBuffer, tarReader, hdr.Size)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	var manifest schema1.SignedManifest
	if err := manifest.UnmarshalJSON(manifestBuffer.Bytes()); err != nil {
		log.Error(err)
		return nil, err
	}

	verifiedManifest, err := verifySchema1Manifest(&manifest, ref)
	if err != nil {
		return nil, err
	}

	fmt.Printf("verifiedManifest = %#v\n", verifiedManifest)

	return verifiedManifest, nil
}

// downloadBlob will download and write the layer to the workDir, in the docker format
func downloadBlobFromTar(tarReader *tar.Reader, blobsum digest.Digest, layerDir string) (layer.DiffID, error) {
	hdr, err := tarReader.Next()
	if err != nil { // EOF is also an error.  We expect a certain number of layers.
		log.Errorf("Cannot read layer: %v", err)
		return layer.DiffID(""), err
	}

	log.Debugf("Expecting %s layer (%d bytes)", blobsum, hdr.Size)

	if hdr.Name != blobsum.String() {
		err := fmt.Errorf("Expected layer %q, but got layer %q", blobsum, hdr.Name)
		log.Error(err)
		return layer.DiffID(""), err
	}

	gzipDigest := digest.Canonical.New()
	responseReader := io.TeeReader(tarReader, gzipDigest.Hash())

	archive, err := gzip.NewReader(responseReader)
	if err != nil {
		log.Errorf("Failed to create gzip reader: %v", err)
		return layer.DiffID(""), err
	}
	defer archive.Close()

	target := filepath.Join(layerDir, "layer.tar")
	writer, err := os.Create(target)
	if err != nil {
		log.Errorf("Failed to create tar file %s: %v", target, err)
		return layer.DiffID(""), err
	}
	defer writer.Close()

	tarDigest := digest.Canonical.New()
	tarWriter := io.MultiWriter(writer, tarDigest.Hash())

	// TODO: Anyway to check we read the right number of bytes from the original tar?
	bytesExtracted, err := io.Copy(tarWriter, archive)
	log.Debugf("Wrote %d bytes to layer %s", bytesExtracted, blobsum)
	if err != nil {
		log.Error(err)
		return layer.DiffID(""), err
	}

	computedBlobsum := digest.Digest(gzipDigest.Digest())
	if blobsum.String() != computedBlobsum.String() {
		err := fmt.Errorf("Downloaded layer blobsum does not match expected blobsum: %s != %s", blobsum, computedBlobsum)
		log.Error(err)
		return layer.DiffID(""), err
	}

	diffID := digest.Digest(tarDigest.Digest())
	log.Debugf("Downloaded layer %s, with blobsum %s", diffID, computedBlobsum)

	return layer.DiffID(diffID), nil
}

func skipLayerInTar(tarReader *tar.Reader, blobsum digest.Digest) error {
	hdr, err := tarReader.Next()
	if err != nil { // EOF is also an error.  We expect a certain number of layers.
		log.Errorf("Cannot read layer: %v", err)
		return err
	}

	if hdr.Name != blobsum.String() {
		err := fmt.Errorf("Expected layer %q, but got layer %q", blobsum, hdr.Name)
		log.Error(err)
		return err
	}

	io.Copy(ioutil.Discard, tarReader)
	return nil
}

// isSupportedProtocol will communicate with the remote server and validate that it supports
// the v2 protocol
func isSupportedProtocol(dockerRemote *DockerRemote) (bool, error) {
	uris := []string{
		fmt.Sprintf("https://%s/%s/", dockerRemote.Hostname, dockerRemote.PreferredProto),
		fmt.Sprintf("https://%s/%s/_ping", dockerRemote.Hostname, dockerRemote.PreferredProto),
	}

	for _, uri := range uris {
		client := requests.GlobalHttpClient()

		req, err := client.NewRequest("GET", uri, nil)
		if err != nil {
			log.Infof("Error pinging URL %q: %v", uri, err)
			continue
		}

		resp, err := doRequest(req, client, dockerRemote, 0)
		if err != nil {
			continue
		}

		if resp.StatusCode == http.StatusOK {
			return true, nil
		}
	}
	return false, nil
}

// getJWTToken will return a new JWT token from the resources in the authenticateHeader string
func getJWTToken(dockerRemote *DockerRemote, authenticateHeader string) error {
	if !strings.HasPrefix(authenticateHeader, "Bearer ") {
		return errors.New("only bearer auth is implemented")
	}
	authenticateHeader = strings.TrimPrefix(authenticateHeader, "Bearer ")

	headerParts := strings.Split(authenticateHeader, ",")
	var realm, scope, service string
	for _, headerPart := range headerParts {
		split := strings.Split(headerPart, "=")
		if len(split) != 2 {
			continue
		}

		switch split[0] {
		case "realm":
			realm = strings.Trim(split[1], "\"")
		case "service":
			service = strings.Trim(split[1], "\"")
		case "scope":
			scope = strings.Trim(split[1], "\"")
		}
	}

	v := url.Values{}
	v.Set("service", service)
	if len(scope) > 0 {
		v.Set("scope", scope)
	}
	uri := fmt.Sprintf("%s?%s", realm, v.Encode())

	fmt.Printf("auth uri = %s\n", uri)
	client := requests.GlobalHttpClient()

	req, err := client.NewRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return err
	}

	if dockerRemote.Username != "" && dockerRemote.Password != "" {
		req.SetBasicAuth(dockerRemote.Username, dockerRemote.Password)
	} else if dockerRemote.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", dockerRemote.Token))
	}

	resp, err := client.Do(req)
	if err != nil {
		log.Error(err)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected response: %d", resp.StatusCode)
		log.Error(err)
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	type tokenResponse struct {
		Token string `json:"token"`
	}
	tr := tokenResponse{}
	if err := json.Unmarshal(body, &tr); err != nil {
		return err
	}

	dockerRemote.ServiceHostname = service
	dockerRemote.JWTToken = tr.Token

	return nil
}

// doRequest will actually make the request, and will authenticate with the v2 auth server, if needed
func doRequest(req *http.Request, client *requests.HttpClient, dockerRemote *DockerRemote, attemptNumber int) (*http.Response, error) {
	if attemptNumber == 3 { // if count is 0 based, 3 attempts will be made
		err := errors.New("Too many retries")
		log.Error(err)
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", dockerRemote.JWTToken))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	// Search for pullV2Tag in docker, look in pull_v1 and pull_v2
	if resp.StatusCode == http.StatusUnauthorized {
		// We need a token and try again...
		if err := getJWTToken(dockerRemote, resp.Header.Get("Www-Authenticate")); err != nil {
			return nil, err
		}

		return doRequest(req, client, dockerRemote, attemptNumber+1)
	}

	return resp, nil
}
