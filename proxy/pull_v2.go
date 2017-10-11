package proxy

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/replicatedcom/harpoon/log"
	"github.com/replicatedcom/harpoon/remote"
)

type Proxy struct {
	Remote *remote.DockerRemote
}

type ManifestResponse struct {
	ManifestId  string
	ContentType string
	SignedJson  []byte
}

func (p *Proxy) GetManifestV2(namespace, imagename, reference string) (*ManifestResponse, error) {
	uri := fmt.Sprintf("https://%s/v2/%s/%s/manifests/%s", p.Remote.Hostname, namespace, imagename, reference)
	log.Debugf("Getting manifest from %s", uri)

	req, err := p.Remote.NewHttpRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// This does not guarantee that we will get manifest v2...
	req.Header.Set("Accept", "application/vnd.docker.distribution.manifest.v2+json")

	resp, err := p.Remote.DoWithRetry(req, 3)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("unexpected status code %d", resp.StatusCode)
		log.Errorf("status=%d; error=%s", resp.StatusCode, body)
		return nil, err
	}

	result := &ManifestResponse{
		ManifestId:  resp.Header.Get("Docker-Content-Digest"),
		ContentType: resp.Header.Get("Content-Type"),
		SignedJson:  body,
	}

	return result, nil
}
