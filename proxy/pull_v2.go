package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/textproto"
	"strconv"

	"github.com/docker/distribution/reference"
	"github.com/pkg/errors"
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
	ContentRange  string
	StatusCode    int
	Header        http.Header
}

func (b *BlobResponse) Close() error {
	if b == nil || b.Reader == nil {
		return nil
	}
	return b.Reader.Close()
}

func (p *Proxy) GetManifestV2(namespace, imagename, ref string, accept []string) (*ManifestResponse, error) {
	var uri string
	// ECR repos are not given a namespace unless the following repo naming convention is followed:
	// `my-example-namespace/my-repo`
	if len(namespace) == 0 {
		uri = fmt.Sprintf("https://%s/v2/%s/manifests/%s", p.Remote.Hostname, imagename, ref)
	} else {
		uri = fmt.Sprintf("https://%s/v2/%s/%s/manifests/%s", p.Remote.Hostname, namespace, imagename, ref)
	}
	log.Debugf("Getting manifest from %s", uri)

	req, err := p.Remote.NewHttpRequest("GET", uri, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}

	// Get manifest schema version requested from client
	if len(accept) > 0 {
		req.Header[textproto.CanonicalMIMEHeaderKey("Accept")] = accept
	}

	log.Debugf("Pulling %s with accept content type: %q", imagename, accept)

	// We can request pull scope in case oauth implementation does not provide scope
	// in the authorization failure.
	additionalScope := fmt.Sprintf("repository:%s:pull", reference.Path(p.Remote.Ref))

	resp, err := p.Remote.DoWithRetry(req, 3, additionalScope)
	if err != nil {
		return nil, errors.Wrap(err, "failed to do request")
	}
	defer resp.Body.Close()

	log.Debugf("Got %s with content type: %q", imagename, resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Wrapf(&ProxyError{
			StatusCode:   resp.StatusCode,
			ResponseBody: body,
			ContentType:  resp.Header.Get("Content-Type"),
		}, "unexpected status code: %d", resp.StatusCode)
	}

	result := &ManifestResponse{
		ManifestId:  resp.Header.Get("Docker-Content-Digest"),
		ContentType: resp.Header.Get("Content-Type"),
		SignedJson:  body,
	}

	return result, nil
}

func (p *Proxy) GetBlobV2(namespace, imagename, digestFull string, additionalHeaders http.Header) (*BlobResponse, error) {
	req, err := p.makeBlobRequest("GET", namespace, imagename, digestFull, additionalHeaders)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to make proxied blob request for %s", req.URL.String())
	}

	resp, err := p.Remote.Do(req)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to do proxied blob request for %s", req.URL.String())
	}

	if resp.StatusCode != http.StatusOK {
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Errorf("failed to read response body: %v", err)
		}
		return nil, errors.Wrapf(&ProxyError{
			StatusCode:   resp.StatusCode,
			ResponseBody: body,
			ContentType:  resp.Header.Get("Content-Type"),
		}, "unexpected status code: %d", resp.StatusCode)
	}

	return p.makeBlobResponse(resp, req.URL.String()), nil
}

func (p *Proxy) makeBlobRequest(httpMethod, namespace, imagename, digestFull string, additionalHeaders http.Header) (*http.Request, error) {
	var uri string
	// ECR repos are not given a namespace unless the following repo naming convention is followed:
	// `my-example-namespace/my-repo`
	if len(namespace) == 0 {
		uri = fmt.Sprintf("https://%s/v2/%s/blobs/%s", p.Remote.Hostname, imagename, digestFull)
	} else {
		uri = fmt.Sprintf("https://%s/v2/%s/%s/blobs/%s", p.Remote.Hostname, namespace, imagename, digestFull)
	}
	log.Debugf("Getting blob from %s", uri)

	req, err := p.Remote.NewHttpRequest(httpMethod, uri, nil)
	if err != nil {
		return nil, err
	}

	for key, vals := range additionalHeaders {
		req.Header[key] = vals
	}

	return req, nil
}

func (p *Proxy) makeBlobResponse(resp *http.Response, uri string) *BlobResponse {
	var contentLength int64
	if resp.Header.Get("Content-Length") == "" {
		log.Warningf("Content length empty for %s", uri)
	} else {
		l, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
		if err != nil {
			log.Warningf("Unknown content length for %s", uri)
		}
		contentLength = l
	}

	result := &BlobResponse{
		Reader:        resp.Body,
		ContentType:   resp.Header.Get("Content-Type"),
		ContentLength: contentLength,
		ContentRange:  resp.Header.Get("Content-Range"),
		StatusCode:    resp.StatusCode,
		Header:        resp.Header,
	}

	return result
}
