package dockerreg

import (
	"github.com/replicatedcom/harpoon/log"

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
		log.Error(err)
		panic(err)
	}
}

func ImportFromRemote(remote *DockerRemote, proxy string) error {
	if err := remote.InitClient(proxy); err != nil {
		return err
	}

	localStore, err := remote.PullImage()
	if localStore != nil {
		defer localStore.delete()
	}

	if err != nil {
		return err
	}

	return ImportFromLocal(localStore)
}

func ImportFromLocal(localStore *v1Store) error {
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
		log.Error(err)
		return err
	}

	return nil
}
