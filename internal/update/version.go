package update

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var semverCoreRe = regexp.MustCompile(`(?i)^v?(\d+)\.(\d+)\.(\d+)`)

type semverCore struct {
	major int
	minor int
	patch int
}

func parseSemverCore(s string) (semverCore, error) {
	s = strings.TrimSpace(s)
	m := semverCoreRe.FindStringSubmatch(s)
	if len(m) != 4 {
		return semverCore{}, fmt.Errorf("invalid semver core: %q", s)
	}
	maj, _ := strconv.Atoi(m[1])
	min, _ := strconv.Atoi(m[2])
	pat, _ := strconv.Atoi(m[3])
	return semverCore{major: maj, minor: min, patch: pat}, nil
}

func compareVersions(a, b string) int {
	va, ea := parseSemverCore(a)
	vb, eb := parseSemverCore(b)
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
