package dockerreg

import (
	"errors"
	"fmt"
	"strings"

	"github.com/replicatedcom/harpoon/log"
	"github.com/replicatedcom/harpoon/requests"

	"github.com/docker/docker/reference"
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

	ServiceHostname string // ServiceHostname is the endpoint we are told to connect to.
	JWTToken        string // JWTToken is the 2.0 scoped token to use for auth, generated by the remote server.
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

	named, err := reference.ParseNamed(imageURI)
	if err != nil {
		log.Error(err)
		return nil, err
	}
	dockerRemote.Ref = named

	return &dockerRemote, nil
}

func (remote *DockerRemote) InitClient(proxy string) error {
	return requests.InitGlobalHttpClient(proxy)
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
