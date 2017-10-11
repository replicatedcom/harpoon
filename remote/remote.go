package remote

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/replicatedcom/harpoon/log"
	"github.com/replicatedcom/harpoon/requests"

	"github.com/docker/distribution/reference"
)

const (
	maxRequestRetries = 3
)

// DockerRemote represents a parsed docker:// image uri.
type DockerRemote struct {
	Hostname  string
	Namespace string
	ImageName string
	Tag       string
	Ref       reference.Named

	PreferredProto string

	Username string
	Password string
	Token    string

	RegistryReader  io.Reader
	ServiceHostname string // ServiceHostname is the endpoint we are told to connect to by the initial call to the registry.
	JWTToken        string // JWTToken is the 2.0 scoped token to use for auth, generated by the remote server.

	RemoteEndpoints string // Data from "X-Docker-Endpoints" header
	RemoteToken     string // Data from "X-Docker-Token" header
	RemoteCookie    string // Data from "Set-Cookie" header

	client *requests.HttpClient
}

const (
	DefaultHostname  = "index.docker.io"
	DefaultNamespace = "library"
	DefaultTag       = "latest"
)

// ParseDockerURI will accept a docker:// image uri and return a DockerRemote or error object.
func ParseDockerURI(imageURI string) (*DockerRemote, error) {
	if !strings.HasPrefix(imageURI, "docker://") {
		return nil, errors.New("invalid image uri - expected docker:// prefix")
	}

	imageURI = strings.TrimPrefix(imageURI, "docker://")

	// The format can vary.  There may be a host and there may be a namespace.  Both are optional.
	// But it's not currently possible to have a host without a namespace.
	// So, valid options are:
	//
	//  host/namespace/image:tag
	//  host/namespace/image
	//  namespace/image:tag
	//  namespace/image
	//  image:tag
	//  image

	imageURIAndTag := strings.Split(imageURI, ":")
	imageURIParts := strings.Split(imageURIAndTag[0], "/")

	dockerRemote := DockerRemote{
		Hostname:  DefaultHostname,
		Namespace: DefaultNamespace,
		Tag:       DefaultTag,
	}

	if len(imageURIParts) == 3 {
		dockerRemote.Hostname = imageURIParts[0]
		dockerRemote.Namespace = imageURIParts[1]
		dockerRemote.ImageName = imageURIParts[2]
	} else if len(imageURIParts) == 2 {
		dockerRemote.Namespace = imageURIParts[0]
		dockerRemote.ImageName = imageURIParts[1]
	} else if len(imageURIParts) == 1 {
		dockerRemote.ImageName = imageURIParts[0]
	} else {
		return nil, errors.New("invalid image uri - expected less than 3 separators (/)")
	}

	if len(imageURIAndTag) == 2 {
		dockerRemote.Tag = imageURIAndTag[1]
	}

	named, err := reference.ParseNormalizedNamed(imageURI)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	dockerRemote.Ref = named

	if err := dockerRemote.InitClient(); err != nil {
		log.Error(err)
		return nil, err
	}

	return &dockerRemote, nil
}

func (remote *DockerRemote) InitClient() error {
	client, err := requests.GetHttpClient(os.Getenv("HTTP_PROXY"))
	if err != nil {
		log.Error(err)
		return err
	}

	remote.client = client
	return nil
}

func (remote *DockerRemote) GetDisplayName() string {
	name := remote.ImageName
	if remote.Namespace != DefaultNamespace {
		name = fmt.Sprintf("%s/%s", remote.Namespace, name)
	}
	if remote.Hostname != DefaultHostname {
		name = fmt.Sprintf("%s/%s", remote.Hostname, name)
	}
	return name
}

// func (remote *DockerRemote) GetHttpClient() *requests.HttpClient {
// 	return remote.client
// }

func (remote *DockerRemote) NewHttpRequest(method, uri string, body io.Reader) (*http.Request, error) {
	return remote.client.NewRequest(method, uri, body)
}

// Do will not attempt to authenticate if server returns a 401 (or any other error)
func (remote *DockerRemote) Do(req *http.Request) (*http.Response, error) {
	return remote.DoWithRetry(req, 1)
}

// DoRequest will actually make the request, and will authenticate with the v2 auth server, if needed
func (remote *DockerRemote) DoWithRetry(req *http.Request, numAttempts int) (*http.Response, error) {
	if numAttempts == 0 {
		err := errors.New("Too many retries")
		log.Error(err)
		return nil, err
	}

	if remote.JWTToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", remote.JWTToken))
	}

	log.Debugf("++++requesting %s", req.URL)
	resp, err := remote.client.Do(req)
	if err != nil {
		return nil, err
	}
	log.Debugf("++++resp headers:%#v", resp.Header)

	// We need to authenticate after attempting a request in order
	// to receive correct authentication instructions.
	if resp.StatusCode == http.StatusUnauthorized && numAttempts > 1 {
		// We need a token and try again...
		if err := remote.getJWTToken(resp.Header.Get("Www-Authenticate")); err != nil {
			return nil, err
		}

		return remote.DoWithRetry(req, numAttempts-1)
	}

	return resp, nil
}
