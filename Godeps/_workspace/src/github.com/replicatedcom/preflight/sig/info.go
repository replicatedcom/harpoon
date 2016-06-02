package sig

import (
	"github.com/replicatedcom/preflight/log"
	"github.com/replicatedcom/preflight/utils"
)

var (
	// FIXME: depends probably needs to rely on more than just the ID
	// if we allow more than one gatherer per ID (slice not map)
	gatherers []Gatherer
)

type SystemInfo struct {
	Platform            string
	PlaformVersion      string
	PlaformVersionID    string
	LinuxDistribution   string
	LinuxDistributionID string
	LinuxKernelVersion  utils.Version
	RootFsPath          string
	Filesystem          []FSEntry
	MemoryBytes         uint64
	CPUCores            uint64
	DockerVersion       utils.Version
	DockerDriverRootDir string
}

type Gatherer interface {
	Gather(SystemInfo, string) (SystemInfo, error)
	ID() string
	Depends() []string
}

func RegisterGatherer(gatherer Gatherer) {
	gatherers = append(gatherers, gatherer)
}

// NOTE: evaluate if we should use https://github.com/cloudfoundry/gosigar

func Gather(rootFsPath string) SystemInfo {
	info := SystemInfo{}
	depends := make(map[string]struct{})
	complete := make(map[int]struct{})
	for i := 0; i < len(gatherers); i++ {
		for j, gatherer := range gatherers {
			if _, ok := complete[j]; ok {
				continue
			} else if !dependenciesMet(gatherer.Depends(), depends) {
				continue
			}
			var err error
			info, err = gatherer.Gather(info, rootFsPath)
			if err != nil {
				log.Errorf("%s: %v", gatherer.ID(), err)
			}
			depends[gatherer.ID()] = struct{}{}
			complete[j] = struct{}{}
		}
		if len(complete) == len(gatherers) {
			break
		}
	}
	if len(complete) != len(gatherers) {
		for i, gatherer := range gatherers {
			if _, ok := complete[i]; !ok {
				log.Errorf("gatherer %s missing some dependencies %q", gatherer.ID(), gatherer.Depends())
			}
		}
	}
	return info
}

func dependenciesMet(depends []string, complete map[string]struct{}) bool {
	for _, dependency := range depends {
		if _, ok := complete[dependency]; !ok {
			return false
		}
	}
	return true
}
