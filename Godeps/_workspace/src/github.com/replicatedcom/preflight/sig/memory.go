package sig

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/replicatedcom/preflight/utils"
)

func init() {
	RegisterGatherer(Memory{})
}

type Memory struct {
}

func (c Memory) Gather(info SystemInfo, rootFsPath string) (SystemInfo, error) {
	var err error
	switch info.Platform {
	case "linux":
		info, err = memoryLinux(info, rootFsPath)
	case "":
		err = errors.New("os empty")
	default:
		err = fmt.Errorf("os unsupported: %s", info.Platform)
	}
	return info, err
}

func (c Memory) ID() string {
	return "memory"
}

func (c Memory) Depends() []string {
	return []string{"platform"}
}

func memoryLinux(info SystemInfo, rootFsPath string) (SystemInfo, error) {
	f, err := os.Open(path.Join(rootFsPath, "/proc/meminfo"))
	if err != nil {
		return info, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	if !scanner.Scan() {
		return info, errors.New("/proc/meminfo empty")
	}

	l := scanner.Text()
	fs := strings.Fields(l)
	if len(fs) != 3 || fs[2] != "kB" {
		return info, errors.New("/proc/meminfo unexpected format")
	}

	kb, err := strconv.ParseUint(fs[1], 10, 64)
	if err != nil {
		return info, err
	}

	info.MemoryBytes = kb * utils.KILOBYTE
	return info, nil
}
