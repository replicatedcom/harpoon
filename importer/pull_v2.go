package importer

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
	"os"
	"path/filepath"
	"strconv"

	"github.com/replicatedcom/harpoon/log"
	"github.com/replicatedcom/harpoon/remote"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
	"github.com/docker/docker/image"
	"github.com/docker/docker/image/v1"
	"github.com/docker/docker/layer"
	digest "github.com/opencontainers/go-digest"
)

const (
	maxRetries       = 3
	ManifestFileName = "_manifest.json"
)

type Importer struct {
	Remote *remote.DockerRemote
}

func (i *Importer) StreamLayers() (io.ReadCloser, error) {
	pipeReader, pipeWriter := io.Pipe()
	go i.writeLayers(pipeWriter)
	return pipeReader, nil
}

func (i *Importer) writeLayers(pipeWriter *io.PipeWriter) {
	tarWriter := tar.NewWriter(pipeWriter)

	var writeError error
	defer func() {
		pipeWriter.CloseWithError(writeError)
		tarWriter.Close()
	}()

	supported, writeError := i.isSupportedProtocol()
	if writeError != nil {
		return
	}
	if !supported {
		writeError = errors.New("Docker registry v2 protocol is not supported by remote")
		return
	}

	i.Remote.JWTToken = ""
	// NOTE: v2 manifests not supported here
	rawManifest, _, writeError := i.GetManifestBytes(schema1.MediaTypeManifest) // schema1.MediaTypeSignedManifest
	if writeError != nil {
		log.Error(writeError)
		return
	}

	writeError = i.writeLayersV1(tarWriter, rawManifest)
	return
}

func (i *Importer) writeLayersV1(tarWriter *tar.Writer, rawManifest []byte) error {
	tarHeader := &tar.Header{
		Name: ManifestFileName,
		// Mode: 0655,
		Size: int64(len(rawManifest)),
	}
	if err := tarWriter.WriteHeader(tarHeader); err != nil {
		log.Error(err)
		return err
	}

	if _, err := tarWriter.Write(rawManifest); err != nil {
		log.Error(err)
		return err
	}

	var manifest schema1.SignedManifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		log.Error(err)
		return err
	}

	// Send layers in reverse because import needs to read them in reverse order
	for j := len(manifest.FSLayers) - 1; j >= 0; j-- {
		layer := manifest.FSLayers[j]

		blobStream, expectedLenght, err := i.getBlobStream(layer.BlobSum)
		if err != nil {
			log.Error(err)
			return err
		}
		defer blobStream.Close() // ok to keep open until func terminates

		tarHeader := &tar.Header{
			Name: layer.BlobSum.String(),
			Size: expectedLenght,
		}
		if err := tarWriter.WriteHeader(tarHeader); err != nil {
			log.Error(err)
			return err
		}

		if _, err := io.Copy(tarWriter, blobStream); err != nil {
			log.Error(err)
			return err
		}
	}

	return nil
}

func (i *Importer) writeLayersV2(tarWriter *tar.Writer, rawManifest []byte) error {
	tarHeader := &tar.Header{
		Name: ManifestFileName,
		// Mode: 0655,
		Size: int64(len(rawManifest)),
	}
	if err := tarWriter.WriteHeader(tarHeader); err != nil {
		log.Error(err)
		return err
	}

	if _, err := tarWriter.Write(rawManifest); err != nil {
		log.Error(err)
		return err
	}

	var manifest schema2.DeserializedManifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		log.Error(err)
		return err
	}

	// Send layers in reverse because import needs to read them in reverse order
	for j := len(manifest.Layers) - 1; j >= 0; j-- {
		layer := manifest.Layers[j]

		blobStream, expectedLenght, err := i.getBlobStream(layer.Digest)
		if err != nil {
			log.Error(err)
			return err
		}
		defer blobStream.Close() // ok to keep open until func terminates

		tarHeader := &tar.Header{
			Name: layer.Digest.String(),
			Size: expectedLenght,
		}
		if err := tarWriter.WriteHeader(tarHeader); err != nil {
			log.Error(err)
			return err
		}

		if _, err := io.Copy(tarWriter, blobStream); err != nil {
			log.Error(err)
			return err
		}
	}

	return nil
}

// ImportFromStream will read manifest and layer data from a single tar stream
func (i *Importer) ImportFromStream(reader io.Reader, imageURI string) error {
	tmpStore, err := streamToTempStore(reader, imageURI)
	if tmpStore != nil {
		defer tmpStore.delete()
	}
	if err != nil {
		return err
	}
	return i.ImportFromLocal(tmpStore)
}

func streamToTempStore(reader io.Reader, imageURI string) (*v1Store, error) {
	ref, err := reference.ParseNormalizedNamed(imageURI)
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

		v1ImageJSON := []byte(verifiedManifest.History[i].V1Compatibility)
		if err := json.Unmarshal(v1ImageJSON, &throwAway); err != nil {
			return localStore, err
		}

		h, err := v1.HistoryFromConfig(v1ImageJSON, throwAway.ThrowAway)
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
			if err := json.Unmarshal(v1ImageJSON, &v1Img); err != nil {
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
		v1ImgJSON, err := json.Marshal(v1Img)
		if err != nil {
			log.Error(err)
			return localStore, err
		}
		if err := ioutil.WriteFile(filepath.Join(layerTempDir, "json"), v1ImgJSON, 0644); err != nil {
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
// unused I THINK
func (i *Importer) PullImage() (*v1Store, error) {
	// Validate that the remote server supports the v2 protocol

	if i.Remote.PreferredProto == "v1" {
		return i.PullImageV1()
	}

	supported, err := i.isSupportedProtocol()
	if err != nil {
		return nil, err
	}
	if !supported {
		return nil, errors.New("Docker registry v2 protocol is not supported by remote")
	}

	// TODO: support for manifest v2
	return i.pullImageV2ManifestV1()
}

// PullImage will pull image from v2 registry with manifest v1
func (i *Importer) pullImageV2ManifestV1() (*v1Store, error) {
	// Ugh, this isn't the right design to use here.
	// But the token will be set from checking the /v2/ endpoint without a scope, which will cause a
	// 401 when trying to pull.
	i.Remote.JWTToken = ""

	verifiedManifest, err := i.GetManifestV1()
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
	for j := len(verifiedManifest.FSLayers) - 1; j >= 0; j-- {
		layer := verifiedManifest.FSLayers[j]

		var throwAway struct {
			ThrowAway bool `json:"throwaway,omitempty"`
		}

		v1ImageJSON := []byte(verifiedManifest.History[j].V1Compatibility)
		if err := json.Unmarshal(v1ImageJSON, &throwAway); err != nil {
			return localStore, err
		}

		h, err := v1.HistoryFromConfig(v1ImageJSON, throwAway.ThrowAway)
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
		diffID, err := i.downloadBlob(blobSum, layerTempDir)
		if err != nil {
			return localStore, err
		}

		if err = ioutil.WriteFile(filepath.Join(layerTempDir, "VERSION"), []byte("1.0"), 0644); err != nil {
			log.Error(err)
			return localStore, err
		}

		rootFS.Append(diffID) // rootFS must contain this layer ID to produce correct chain ID
		v1Img := image.V1Image{}
		if j == 0 {
			if err := json.Unmarshal(v1ImageJSON, &v1Img); err != nil {
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
		v1ImgJSON, err := json.Marshal(v1Img)
		if err != nil {
			log.Error(err)
			return localStore, err
		}
		if err := ioutil.WriteFile(filepath.Join(layerTempDir, "json"), v1ImgJSON, 0644); err != nil {
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

	if err := localStore.writeRepositoriesFile(i.Remote.Ref, imageID); err != nil {
		return localStore, err
	}

	if err := localStore.writeManifestFile(i.Remote.Ref, imageID, layerV1IDs); err != nil {
		return localStore, err
	}

	return localStore, nil
}

// downloadBlob will download and write the layer to the workDir, in the docker format
func (i *Importer) downloadBlob(blobsum digest.Digest, layerDir string) (layer.DiffID, error) {
	uri := fmt.Sprintf("https://%s/v2/%s/%s/blobs/%s", i.Remote.Hostname, i.Remote.Namespace, i.Remote.ImageName, blobsum.String())

	log.Debugf("Downloading blob from %q\n", uri)

	req, err := i.Remote.NewHttpRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return layer.DiffID(""), err
	}

	resp, err := i.Remote.DoWithRetry(req, maxRetries)
	if err != nil {
		return layer.DiffID(""), err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("Unexpected status code for %s: %d", uri, resp.StatusCode)
		log.Error(err)
		return layer.DiffID(""), err
	}

	gzipDigest := digest.Canonical.Digester()
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

	tarDigest := digest.Canonical.Digester()
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

func (i *Importer) getBlobStream(blobsum digest.Digest) (io.ReadCloser, int64, error) {
	uri := fmt.Sprintf("https://%s/v2/%s/%s/blobs/%s", i.Remote.Hostname, i.Remote.Namespace, i.Remote.ImageName, blobsum.String())

	log.Debugf("Downloading blob from %q", uri)

	req, err := i.Remote.NewHttpRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return nil, 0, err
	}

	resp, err := i.Remote.DoWithRetry(req, maxRetries)
	if err != nil {
		return nil, 0, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		err := fmt.Errorf("Unexpected status code for %s: %d", uri, resp.StatusCode)
		log.Error(err)
		return nil, 0, err
	}

	log.Debugf("Responded with content-length: %q", resp.Header.Get("Content-Length"))
	expectedSize, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		log.Error(err)
		expectedSize = -1
	}

	return resp.Body, expectedSize, nil
}

// getManifest will return the remote manifest for the image.
func (i *Importer) GetManifestV1() (*schema1.Manifest, error) {
	rawManifest, _, err := i.GetManifestBytes(schema1.MediaTypeManifest) // schema1.MediaTypeSignedManifest
	if err != nil {
		return nil, err
	}

	var manifest schema1.SignedManifest
	if err := json.Unmarshal(rawManifest, &manifest); err != nil {
		log.Error(err)
		return nil, err
	}

	log.Debugf("manifest = %#v", manifest)

	verifiedManifest, err := verifySchema1Manifest(&manifest, i.Remote.Ref)
	if err != nil {
		return nil, err
	}

	log.Debugf("verifiedManifest = %#v", verifiedManifest)

	return verifiedManifest, nil
}

func (i *Importer) GetManifestBytes(mediaTypes ...string) ([]byte, string, error) {
	uri := fmt.Sprintf("https://%s/v2/%s/%s/manifests/%s", i.Remote.Hostname, i.Remote.Namespace, i.Remote.ImageName, i.Remote.Tag)

	req, err := i.Remote.NewHttpRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return nil, "", err
	}

	if len(mediaTypes) == 0 {
		mediaTypes = []string{
			schema1.MediaTypeManifest,
			schema1.MediaTypeSignedManifest,
			schema2.MediaTypeManifest,
		}
	}

	for _, mediaType := range mediaTypes {
		req.Header.Set("Accept", mediaType)
	}

	log.Debugf("Get manifest %s", uri)

	// We can request pull scope in case oauth implementation does not provide scope
	// in the authorization failure.
	additionalScope := fmt.Sprintf("repository:%s:pull", reference.Path(i.Remote.Ref))

	resp, err := i.Remote.DoWithRetry(req, maxRetries, additionalScope)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("Unexpected status code for %s: %d", uri, resp.StatusCode)
		log.Error(err)
		return nil, "", err
	}

	mediaType := resp.Header.Get("Content-Type")
	log.Debugf("Responded with media-type: %q", mediaType)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, mediaType, err
	}

	return body, mediaType, nil
}

func getManifestFromTar(tarReader *tar.Reader, ref reference.Named) (*schema1.Manifest, error) {
	hdr, err := tarReader.Next()
	if err != nil { // EOF is also an error here.  We need the manifest.
		err := fmt.Errorf("Cannot read manifest from tar stream: %v", err)
		log.Error(err)
		return nil, err
	}

	if hdr.Name != ManifestFileName {
		err := fmt.Errorf("Expected %q but found %q", ManifestFileName, hdr.Name)
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
	if err := json.Unmarshal(manifestBuffer.Bytes(), &manifest); err != nil {
		log.Error(err)
		return nil, err
	}

	verifiedManifest, err := verifySchema1Manifest(&manifest, ref)
	if err != nil {
		return nil, err
	}

	log.Debugf("verifiedManifest = %#v", verifiedManifest)

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

	gzipDigest := digest.Canonical.Digester()
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

	tarDigest := digest.Canonical.Digester()
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
func (i *Importer) isSupportedProtocol() (bool, error) {
	uris := []string{
		fmt.Sprintf("https://%s/%s/", i.Remote.Hostname, i.Remote.PreferredProto),
		fmt.Sprintf("https://%s/%s/_ping", i.Remote.Hostname, i.Remote.PreferredProto),
	}

	for _, uri := range uris {
		req, err := i.Remote.NewHttpRequest("GET", uri, nil)
		if err != nil {
			log.Infof("Error pinging URL %q: %v", uri, err)
			continue
		}

		resp, err := i.Remote.DoWithRetry(req, maxRetries)
		if err != nil {
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			return true, nil
		}
	}
	return false, nil
}
