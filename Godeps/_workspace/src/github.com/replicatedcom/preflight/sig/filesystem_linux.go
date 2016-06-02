package sig

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/replicatedcom/preflight/log"

	"golang.org/x/sys/unix"
)

type linuxFSTabEntry struct {
	Spec    string
	File    string
	Vfstype string
	Mntops  []string
	Freq    int
	Passno  int
}

func filesystemLinux(info SystemInfo, rootFsPath string) (SystemInfo, error) {
	fsTabEntries, err := parseProcMounts(rootFsPath)
	if err != nil {
		return info, err
	}

	for _, fsTabEntry := range fsTabEntries {
		total, free, err := fsStatisticsLinux(fsTabEntry.File)
		if err != nil {
			log.Warningf("Failed to get filesystem status for %s: %v", fsTabEntry.File, err)
			continue
		}
		info.Filesystem = append(info.Filesystem, FSEntry{fsTabEntry.File, total, free})
	}

	return info, nil
}

func parseProcMounts(rootFsPath string) ([]*linuxFSTabEntry, error) {
	f, err := os.Open(filepath.Join(rootFsPath, "/proc/mounts"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	results := []*linuxFSTabEntry{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) != 6 {
			return nil, errors.New("/proc/mounts unexpected format")
		}

		freq, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, err
		}
		passno, err := strconv.Atoi(fields[5])
		if err != nil {
			return nil, err
		}
		results = append(results, &linuxFSTabEntry{
			Spec:    fields[0],
			File:    fields[1],
			Vfstype: fields[2],
			Mntops:  strings.Split(fields[3], ","),
			Freq:    freq,
			Passno:  passno,
		})
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return results, nil
}

func fsStatisticsLinux(filePath string) (total uint64, avail uint64, err error) {
	var stat unix.Statfs_t
	err = unix.Statfs(filePath, &stat)
	if err != nil {
		return
	}
	total = stat.Blocks * uint64(stat.Bsize)
	avail = stat.Bavail * uint64(stat.Bsize)
	return
}
