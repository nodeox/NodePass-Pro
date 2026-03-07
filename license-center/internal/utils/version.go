package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var semverPattern = regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)(.*)$`)

type semver struct {
	Major  int
	Minor  int
	Patch  int
	Suffix string
}

func parseSemver(value string) semver {
	trimmed := strings.TrimSpace(strings.TrimPrefix(strings.TrimPrefix(value, "v"), "V"))
	m := semverPattern.FindStringSubmatch(trimmed)
	if len(m) != 5 {
		return semver{Suffix: trimmed}
	}
	major, _ := strconv.Atoi(m[1])
	minor, _ := strconv.Atoi(m[2])
	patch, _ := strconv.Atoi(m[3])
	return semver{Major: major, Minor: minor, Patch: patch, Suffix: m[4]}
}

// CompareVersion 比较版本号。
// 返回值: -1 左小于右, 0 相等, 1 左大于右。
func CompareVersion(left, right string) int {
	l := parseSemver(left)
	r := parseSemver(right)
	if l.Major != r.Major {
		if l.Major < r.Major {
			return -1
		}
		return 1
	}
	if l.Minor != r.Minor {
		if l.Minor < r.Minor {
			return -1
		}
		return 1
	}
	if l.Patch != r.Patch {
		if l.Patch < r.Patch {
			return -1
		}
		return 1
	}
	if l.Suffix == r.Suffix {
		return 0
	}
	if l.Suffix == "" {
		return 1
	}
	if r.Suffix == "" {
		return -1
	}
	if l.Suffix < r.Suffix {
		return -1
	}
	if l.Suffix > r.Suffix {
		return 1
	}
	return 0
}

// CheckVersionRange 校验版本是否在区间内。
func CheckVersionRange(current, minVersion, maxVersion string) error {
	if strings.TrimSpace(minVersion) != "" && CompareVersion(current, minVersion) < 0 {
		return fmt.Errorf("版本过低: current=%s, min=%s", current, minVersion)
	}
	if strings.TrimSpace(maxVersion) != "" && CompareVersion(current, maxVersion) > 0 {
		return fmt.Errorf("版本过高: current=%s, max=%s", current, maxVersion)
	}
	return nil
}
