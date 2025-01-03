package importer

import (
	"github.com/pkg/errors"
	"github.com/replicatedcom/harpoon/log"
	"github.com/replicatedcom/harpoon/remote"

	"github.com/docker/docker/pkg/archive"
	docker "github.com/fsouza/go-dockerclient"
)

var (
	dockerClient *docker.Client
)

func init() {
	var err error
	dockerClient, err = docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		panic(errors.Wrap(err, "failed to create docker client"))
	}
}

// ImportFromRemote imports an image into the store from a remote repo.
// unused I THINK
func ImportFromRemote(dockerRemote *remote.DockerRemote) error {
	i := &Importer{Remote: dockerRemote}

	localStore, err := i.PullImage()
	if localStore != nil {
		defer localStore.delete()
	}

	if err != nil {
		return err
	}

	return i.ImportFromLocal(localStore)
}

func (i *Importer) ImportFromLocal(localStore *v1Store) error {
	log.Debugf("Loading image from %s", localStore.Workspace)

	archive, err := archive.TarWithOptions(localStore.Workspace, &archive.TarOptions{Compression: archive.Uncompressed})
	if err != nil {
		return err
	}
	defer archive.Close()

	loadImageOptions := docker.LoadImageOptions{
		InputStream: archive,
	}
	if err = dockerClient.LoadImage(loadImageOptions); err != nil {
		return errors.Wrap(err, "failed to load image")
	}

	return nil
}
