package sig

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path"
	"strings"

	"github.com/replicatedcom/preflight/log"
	"github.com/replicatedcom/preflight/utils"

	"github.com/blang/semver"
)

func init() {
	RegisterGatherer(KernelVersion{})
}

type KernelVersion struct {
}

func (c KernelVersion) Gather(info SystemInfo, rootFsPath string) (SystemInfo, error) {
	var err error
	switch info.Platform {
	case "linux":
		info.LinuxKernelVersion, err = kernelVersionLinux(rootFsPath)
	case "":
		log.Debugf("Failed to detect kernel version: os empty")
	default:
		log.Debugf("Failed to detect kernel version: os unsupported %s", info.Platform)
	}
	return info, err
}

func (c KernelVersion) ID() string {
	return "kernel-version"
}

func (c KernelVersion) Depends() []string {
	return []string{"platform"}
}

func kernelVersionLinux(rootFsPath string) (version utils.Version, err error) {
	var contents []byte
	contents, err = ioutil.ReadFile(path.Join(rootFsPath, "/proc/version"))
	if err != nil {
		return
	}
	parts := strings.SplitN(string(contents), " ", 4)
	if len(parts) < 3 {
		err = errors.New("/proc/version unexpected format")
		return
	}

	var major, minor, patch, parsed int
	var partial string

	parsed, _ = fmt.Sscanf(parts[2], "%d.%d%s", &major, &minor, &partial)
	if parsed < 2 {
		err = fmt.Errorf("failed to parse kernel version %s", parts[2])
		return
	}
	parsed, _ = fmt.Sscanf(partial, ".%d%s", &patch, &partial)

	version.VersionStr = parts[2]
	version.Version, err = semver.Parse(fmt.Sprintf("%d.%d.%d", major, minor, patch))
	return
}
