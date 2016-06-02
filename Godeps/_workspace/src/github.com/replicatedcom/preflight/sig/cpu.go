package sig

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"
)

func init() {
	RegisterGatherer(CPU{})
}

type CPU struct {
}

func (c CPU) Gather(info SystemInfo, rootFsPath string) (SystemInfo, error) {
	var err error
	switch info.Platform {
	case "linux":
		info, err = cpuLinux(info, rootFsPath)
	case "":
		err = errors.New("os empty")
	default:
		err = fmt.Errorf("os unsupported: %s", info.Platform)
	}
	return info, err
}

func (c CPU) ID() string {
	return "cpu"
}

func (c CPU) Depends() []string {
	return []string{"platform"}
}

func cpuLinux(info SystemInfo, rootFsPath string) (SystemInfo, error) {
	f, err := os.Open(path.Join(rootFsPath, "/proc/cpuinfo"))
	if err != nil {
		return info, err
	}
	defer f.Close()

	var procs uint64

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		l := scanner.Text()
		if strings.Contains(l, "processor") {
			fs := strings.Fields(l)
			if len(fs) != 3 {
				return info, errors.New("/proc/cpuinfo unexpected format")
			}
			var err error
			procs, err = strconv.ParseUint(fs[2], 10, 64)
			if err != nil {
				return info, err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return info, err
	}

	if procs > 0 {
		info.CPUCores = procs + 1
	}

	return info, nil
}
