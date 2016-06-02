package utils

import (
	"strings"

	"github.com/blang/semver"
)

type Version struct {
	Version    semver.Version
	VersionStr string
}

func (v *Version) IsZero() bool {
	return v.Version.EQ(semver.Version{})
}

func (v Version) String() string {
	if len(v.VersionStr) > 0 {
		return v.VersionStr
	}
	return v.Version.String()
}

func SemverParse(s string) (semver.Version, error) {
	v, err := semver.Parse(s)
	if err == nil || err.Error() != "No Major.Minor.Patch elements found" {
		return v, err
	}
	parts := strings.SplitN(s, ".", 3)
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	s = strings.Join(parts, ".")
	return semver.Parse(s)
}
