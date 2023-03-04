package db_migrator

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	PreRelease int
}

func (v Version) String() string {
	return strconv.Itoa(v.Major) + "." +
		strconv.Itoa(v.Minor) + "." +
		strconv.Itoa(v.Patch) + "." +
		strconv.Itoa(v.PreRelease)
}

func (v Version) Equals(version Version) bool {
	return v == version
}

func (v Version) MoreThan(version Version) bool {
	if v.Major > version.Major {
		return true
	} else if v.Major < version.Major {
		return false
	}

	if v.Minor > version.Minor {
		return true
	} else if v.Minor < version.Minor {
		return false
	}

	if v.Patch > version.Patch {
		return true
	} else if v.Patch < version.Patch {
		return false
	}

	if v.PreRelease > version.PreRelease {
		return true
	} else if v.PreRelease < version.PreRelease {
		return false
	}

	return false
}

func (v Version) MoreOrEqual(version Version) bool {
	return v.MoreThan(version) || v.Equals(version)
}

func (v Version) LessThan(version Version) bool {
	if v.Major < version.Major {
		return true
	} else if v.Major > version.Major {
		return false
	}

	if v.Minor < version.Minor {
		return true
	} else if v.Minor > version.Minor {
		return false
	}

	if v.Patch < version.Patch {
		return true
	} else if v.Patch > version.Patch {
		return false
	}

	if v.PreRelease < version.PreRelease {
		return true
	} else if v.PreRelease > version.PreRelease {
		return false
	}

	return false
}

func (v Version) LessOrEqual(version Version) bool {
	return v.LessThan(version) || v.Equals(version)
}

func parseVersion(versionString string) (Version, error) {
	re := regexp.MustCompile(
		`^(?:\D*)?(?P<major>\d+)\.(?P<minor>\d+)\.(?P<patch>\d+)(?:\.(?P<pre_release>\d+))?(?:.*)?$`,
	)
	match := re.FindStringSubmatch(strings.TrimSpace(versionString))
	if match == nil {
		return Version{}, errors.New("version parse failed")
	}

	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	patch, _ := strconv.Atoi(match[3])
	preRelease, _ := strconv.Atoi(match[4])
	return Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		PreRelease: preRelease,
	}, nil
}

func mustParseVersion(versionString string) Version {
	v, err := parseVersion(versionString)
	if err != nil {
		panic(err)
	}
	return v
}
