package v1

import (
	"errors"

	"github.com/replicatedhq/harpoon/dockerreg"
)

// PullImage will pull the v1 image `image`, using the proxy server at `proxy`, and will
// ignore the cache if `force` is `true`.
func PullImage(dockerRemote *dockerreg.DockerRemote, proxy string, force bool) error {
	return errors.New("not implemented")
}
