package proxy

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"

	"github.com/docker/distribution/manifest/schema2"
	"github.com/docker/distribution/reference"
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

type BlobResponse struct {
	Reader        io.ReadCloser
	ContentType   string
	ContentLength int64
}

func (b *BlobResponse) Close() error {
	if b == nil || b.Reader == nil {
		return nil
	}
	return b.Reader.Close()
}

func (p *Proxy) GetManifestV2(namespace, imagename, ref string) (*ManifestResponse, error) {
	var uri string
	if len(namespace) == 0 {
		uri = fmt.Sprintf("https://%s/v2/%s/manifests/%s", p.Remote.Hostname, imagename, ref)
	} else {
		uri = fmt.Sprintf("https://%s/v2/%s/%s/manifests/%s", p.Remote.Hostname, namespace, imagename, ref)
	}
	log.Debugf("Getting manifest from %s", uri)

	req, err := p.Remote.NewHttpRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	// This does not guarantee that we will get manifest v2...
	req.Header.Set("Accept", schema2.MediaTypeManifest)

	// We can request pull scope in case oauth implementation does not provide scope
	// in the authorization failure.
	additionalScope := fmt.Sprintf("repository:%s:pull", reference.Path(p.Remote.Ref))

	resp, err := p.Remote.DoWithRetry(req, 3, additionalScope)
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

func (p *Proxy) GetBlobV2(namespace, imagename, digestFull string) (*BlobResponse, error) {
	var uri string
	if len(namespace) == 0 {
		uri = fmt.Sprintf("https://%s/v2/%s/blobs/%s", p.Remote.Hostname, imagename, digestFull)
	} else {
		uri = fmt.Sprintf("https://%s/v2/%s/%s/blobs/%s", p.Remote.Hostname, namespace, imagename, digestFull)
	}
	log.Debugf("Getting blob from %s", uri)

	req, err := p.Remote.NewHttpRequest("GET", uri, nil)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	resp, err := p.Remote.DoWithRetry(req, 3)
	if err != nil {
		log.Error(err)
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		err := fmt.Errorf("unexpected status code %d", resp.StatusCode)
		return nil, err
	}

	contentLength, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		log.Warningf("Unknown response size for %s", uri)
	}

	result := &BlobResponse{
		Reader:        resp.Body,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: contentLength,
	}

	return result, nil
}
