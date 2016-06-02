package sig

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"regexp"
)

var (
	systemReleaseRegexp = regexp.MustCompile(`^(.+) release (((?:\d+\.?)+).*)$`)
)

func init() {
	RegisterGatherer(Platform{})
}

type Platform struct {
}

func (c Platform) Gather(info SystemInfo, rootFsPath string) (SystemInfo, error) {
	var err error
	info, err = parseOSRelease(info, rootFsPath)
	if err != nil {
		info, err = parseSystemRelease(info, rootFsPath)
	}
	return info, err
}

func (c Platform) ID() string {
	return "platform"
}

func (c Platform) Depends() []string {
	return nil
}

func parseOSRelease(info SystemInfo, rootFsPath string) (SystemInfo, error) {
	contents, err := ioutil.ReadFile(path.Join(rootFsPath, "/etc/os-release"))
	if err != nil {
		return info, err
	}

	osRelease, err := NewOSRelease(contents)
	if err != nil {
		return info, err
	}

	info.Platform = "linux" // TODO: is this so?
	info.PlaformVersion = osRelease.Version
	info.PlaformVersionID = osRelease.VersionID
	info.LinuxDistribution = osRelease.Name
	info.LinuxDistributionID = osRelease.ID
	return info, nil
}

func parseSystemRelease(info SystemInfo, rootFsPath string) (SystemInfo, error) {
	var err error
	filePaths := []string{"/etc/system-release", "/etc/centos-release", "/etc/redhat-release"}
	for _, filePath := range filePaths {
		err = func() error {
			f, err := os.Open(path.Join(rootFsPath, filePath))
			if err != nil {
				return err
			}
			defer f.Close()

			scanner := bufio.NewScanner(f)
			if !scanner.Scan() {
				return fmt.Errorf("%s empty", filePath)
			}

			matches := systemReleaseRegexp.FindStringSubmatch(scanner.Text())
			if len(matches) == 0 {
				return fmt.Errorf("%s parse error", filePath)
			}

			info.Platform = "linux" // TODO: is this so?
			info.PlaformVersion = matches[2]
			info.PlaformVersionID = matches[3]
			info.LinuxDistribution = matches[1]
			info.LinuxDistributionID = getLinuxDistributionIDFromName(info.LinuxDistribution)
			return nil
		}()
		if err == nil {
			break
		}
	}
	return info, err
}

func getLinuxDistributionIDFromName(name string) string {
	switch name {
	case "Amazon Linux AMI":
		return "amzn"
	case "CentOS", "CentOS Linux":
		return "centos"
	case "Red Hat Enterprise Linux Server":
		return "rhel"
	default:
		return ""
	}
}
