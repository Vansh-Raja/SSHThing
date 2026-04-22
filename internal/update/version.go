package update

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var semverFullRe = regexp.MustCompile(`(?i)^v?(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$`)

type semverVersion struct {
	major      int
	minor      int
	patch      int
	prerelease string
}

func parseSemverVersion(s string) (semverVersion, error) {
	s = strings.TrimSpace(s)
	m := semverFullRe.FindStringSubmatch(s)
	if len(m) != 5 {
		return semverVersion{}, fmt.Errorf("invalid semver: %q", s)
	}
	maj, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	pat, _ := strconv.Atoi(m[3])
	return semverVersion{
		major:      maj,
		minor:      min,
		patch:      pat,
		prerelease: strings.TrimSpace(m[4]),
	}, nil
}

func compareVersions(a, b string) int {
	va, ea := parseSemverVersion(a)
	vb, eb := parseSemverVersion(b)
	if ea != nil || eb != nil {
		return strings.Compare(strings.TrimSpace(a), strings.TrimSpace(b))
	}
	if va.major != vb.major {
		if va.major < vb.major {
			return -1
		}
		return 1
	}
	if va.minor != vb.minor {
		if va.minor < vb.minor {
			return -1
		}
		return 1
	}
	if va.patch != vb.patch {
		if va.patch < vb.patch {
			return -1
		}
		return 1
	}
	return comparePrerelease(va.prerelease, vb.prerelease)
}

func comparePrerelease(a, b string) int {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == b {
		return 0
	}
	if a == "" {
		return 1
	}
	if b == "" {
		return -1
	}

	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")
	for i := 0; i < len(aParts) || i < len(bParts); i++ {
		if i >= len(aParts) {
			return -1
		}
		if i >= len(bParts) {
			return 1
		}
		if aParts[i] == bParts[i] {
			continue
		}

		aNum, aErr := strconv.Atoi(aParts[i])
		bNum, bErr := strconv.Atoi(bParts[i])
		switch {
		case aErr == nil && bErr == nil:
			if aNum < bNum {
				return -1
			}
			return 1
		case aErr == nil:
			return -1
		case bErr == nil:
			return 1
		default:
			if aParts[i] < bParts[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}

func normalizeVersionString(v string) string {
	v = strings.TrimSpace(v)
	if strings.EqualFold(v, "dev") || v == "" {
		return v
	}
	if strings.HasPrefix(strings.ToLower(v), "v") {
		return v[1:]
	}
	return v
}
