package sig

import (
	"path"

	"github.com/blang/semver"
	docker "github.com/fsouza/go-dockerclient"
)

type Docker struct {
	Client *docker.Client
}

func (c Docker) Gather(info SystemInfo, rootFsPath string) (SystemInfo, error) {
	var retErr error
	dockerInfo, err := c.Client.Info()
	if err == nil {
		info, retErr = parseDockerInfo(dockerInfo, info, rootFsPath)
	} else {
		retErr = err
	}
	if info.DockerVersion.IsZero() {
		env, err := c.Client.Version()
		if err == nil {
			info, retErr = parseDockerEnv(env, info, rootFsPath)
		} else {
			retErr = err
		}
	}
	return info, retErr
}

func (c Docker) ID() string {
	return "docker"
}

func (c Docker) Depends() []string {
	return nil
}

func parseDockerInfo(dockerInfo *docker.DockerInfo, info SystemInfo, rootFsPath string) (SystemInfo, error) {
	var err error
	if len(dockerInfo.ServerVersion) > 0 {
		info.DockerVersion.VersionStr = dockerInfo.ServerVersion
		info.DockerVersion.Version, err = semver.Parse(dockerInfo.ServerVersion)
	}
	for _, status := range dockerInfo.DriverStatus {
		// TODO: other graph drivers
		// aufs only
		if status[0] == "Root Dir" {
			info.DockerDriverRootDir = path.Join(rootFsPath, status[1])
			break
		}
	}
	return info, err
}

func parseDockerEnv(env *docker.Env, info SystemInfo, rootFsPath string) (SystemInfo, error) {
	var err error
	info.DockerVersion.VersionStr = env.Get("Version")
	info.DockerVersion.Version, err = semver.Parse(env.Get("Version"))
	return info, err
}
