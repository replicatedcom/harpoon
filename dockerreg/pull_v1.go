package dockerreg

import (
	"errors"
)

// PullImageV1 will pull image from v1 (todo), using the proxy server at `proxy`, and will
// ignore the cache if `force` is `true`.
// TODO: Add auth info to args
func (dockerRemote *DockerRemote) PullImageV1(proxy string, force bool) error {
	return errors.New("Pulling from v1 is not implemented")
}