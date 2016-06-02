package sig

import (
	"errors"
	"fmt"
)

func init() {
	RegisterGatherer(Filesystem{})
}

type Filesystem struct {
}

type FSEntry struct {
	FileSystem string
	TotalBytes uint64
	AvailBytes uint64
}

func (c Filesystem) Gather(info SystemInfo, rootFsPath string) (SystemInfo, error) {
	info.RootFsPath = rootFsPath

	var err error
	switch info.Platform {
	case "linux":
		info, err = filesystemLinux(info, rootFsPath)
		if err == nil {
			info, err = filesystemLinux(info, "/")
		}
	case "":
		err = errors.New("os empty")
	default:
		err = fmt.Errorf("os unsupported: %s", info.Platform)
	}
	return info, err
}

func (c Filesystem) ID() string {
	return "filesystem"
}

func (c Filesystem) Depends() []string {
	return []string{"platform"}
}
