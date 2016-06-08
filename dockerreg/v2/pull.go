package v2

import (
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
	"strings"

	"github.com/replicatedhq/harpoon/dockerreg"
	"github.com/replicatedhq/harpoon/requests"
)

// PullImage will pull the v2 image `image`, using the proxy server at `proxy`, and will
// ignore the cache if `force` is `true`.
func PullImage(dockerRemote *dockerreg.DockerRemote, proxy string, force bool, token string) error {
	// Validate that the remote server supports the v2 protocol
	supported, err := isSupportedProtocol(dockerRemote)
	if err != nil {
		return err
	}
	if !supported {
		return errors.New("Docker registry v2 protocol is not supported by remote")
	}

	// Ugh, this isn't the right design to use here.
	// But the token will be set from checking the /v2/ endpoint without a scope, which will cause a
	// 401 when trying to pull.
	dockerRemote.JWTToken = ""

	// Create a temp directory for this image
	workDir, err := ioutil.TempDir("", "harpoon")
	if err != nil {
		return err
	}

	manifest, err := getManifest(dockerRemote)
	if err != nil {
		return err
	}

	for i, layer := range manifest.FSLayers {
		// Download the layer data
		if err = downloadBlob(dockerRemote, layer.BlobSum, manifest, workDir); err != nil {
			return err
		}

		imageID := string(layer.BlobSum[len(layer.BlobSum)-64:])

		// Write the VERSION file
		if err = ioutil.WriteFile(filepath.Join(workDir, imageID, "VERSION"), []byte("1.0"), 0644); err != nil {
			return err
		}

		// Write the json file
		if err := ioutil.WriteFile(filepath.Join(workDir, imageID, "json"), []byte(manifest.History[i].Compatibility), 0644); err != nil {
			return err
		}

	}

	return errors.New("v2.PullImage done, but importing is not implemented")
}

// downloadBlob will download and write the layer to the workDir, in the docker format
func downloadBlob(dockerRemote *dockerreg.DockerRemote, digest string, manifest *Manifest, workDir string) error {
	uri := fmt.Sprintf("https://%s/v2/%s/%s/blobs/%s", dockerRemote.Hostname, dockerRemote.Namespace, dockerRemote.ImageName, digest)

	fmt.Printf("Downloading blob from %q\n", uri)

	client, err := requests.NewHttpClient("Harpoon-Client/0_0", "", "")
	if err != nil {
		return err
	}

	req, err := client.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}

	resp, err := doRequest(req, client, dockerRemote)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return err
	}

	imageID := string(digest[len(digest)-64:])

	if err = os.MkdirAll(filepath.Join(workDir, imageID), 0644); err != nil {
		return err
	}

	if err = ioutil.WriteFile(filepath.Join(workDir, imageID, "layer.tar.gz"), body, 0644); err != nil {
		return err
	}

	// Ungzip the file
	reader, err := os.Open(filepath.Join(workDir, imageID, "layer.tar.gz"))
	if err != nil {
		return err
	}
	defer reader.Close()

	archive, err := gzip.NewReader(reader)
	if err != nil {
		return err
	}
	defer archive.Close()

	target := filepath.Join(workDir, imageID, "layer.tar")
	writer, err := os.Create(target)
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, archive)
	if err != nil {
		return err
	}

	// TODO verify the digest.  this is completely useless without verification.

	// Delete the gzip
	if err = os.Remove(filepath.Join(workDir, imageID, "layer.tar.gz")); err != nil {
		return err
	}

	return nil
}

// getManifest will return the remote manifest for the image.
func getManifest(dockerRemote *dockerreg.DockerRemote) (*Manifest, error) {
	uri := fmt.Sprintf("https://%s/v2/%s/%s/manifests/%s", dockerRemote.Hostname, dockerRemote.Namespace, dockerRemote.ImageName, dockerRemote.Tag)

	client, err := requests.NewHttpClient("Harpoon-Client/0_0", "", "")
	if err != nil {
		return nil, err
	}

	req, err := client.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := doRequest(req, client, dockerRemote)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Responded with content-type: %q\n", resp.Header.Get("Content-Type"))

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}

	fmt.Printf("body = %s\n", body)
	manifest := Manifest{}
	if err := json.Unmarshal(body, &manifest); err != nil {
		return nil, err
	}
	fmt.Printf("manifest = %#v\n", manifest)

	return &manifest, nil
}

// isSupportedProtocol will communicate with the remote server and validate that it supports
// the v2 protocol
func isSupportedProtocol(dockerRemote *dockerreg.DockerRemote) (bool, error) {
	uri := fmt.Sprintf("https://%s/v2/", dockerRemote.Hostname)

	client, err := requests.NewHttpClient("Harpoon-Client/0_0", "", "")
	if err != nil {
		return false, err
	}

	req, err := client.NewRequest("GET", uri, nil)
	if err != nil {
		return false, err
	}

	resp, err := doRequest(req, client, dockerRemote)
	if err != nil {
		return false, err
	}

	return resp.StatusCode == http.StatusOK, nil
}

// getJWTToken will return a new JWT token from the resources in the authenticateHeader string
func getJWTToken(dockerRemote *dockerreg.DockerRemote, authenticateHeader string) error {
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
	client, err := requests.NewHttpClient("Harpoon-Client/0_0", "", "")
	if err != nil {
		return err
	}

	resp, err := client.Get(uri)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected response: %d", resp.StatusCode)
	}

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
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
func doRequest(req *http.Request, client *requests.HttpClient, dockerRemote *dockerreg.DockerRemote) (*http.Response, error) {
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", dockerRemote.JWTToken))
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		// TODO this will cause unecessary failure when a token is expired.
		// But the idea here is to prevent a stack overflow.
		if dockerRemote.JWTToken != "" {
			fmt.Printf("dockerRemote.JWTToken = %s\n", dockerRemote.JWTToken)
			return nil, errors.New("auth failed with existing jwt token")
		}

		// We need a token and try again...
		if err := getJWTToken(dockerRemote, resp.Header.Get("Www-Authenticate")); err != nil {
			return nil, err
		}

		return doRequest(req, client, dockerRemote)
	}

	return resp, nil
}
