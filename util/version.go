package util

import (
	"regexp"
	"strconv"
	"strings"
)

var versionPattern = regexp.MustCompile(`^\d{1,4}(\.\d{1,4}){0,3}$`)

// IsValidVersion accepts dotted numeric versions such as "1", "1.2" or "1.2.3".
func IsValidVersion(v string) bool {
	return versionPattern.MatchString(strings.TrimSpace(v))
}

// CompareVersions returns -1, 0 or 1 comparing dotted numeric versions
// segment by segment ("1.10.0" > "1.9.3"); missing segments count as 0.
// Anything unparsable sorts below every valid version.
func CompareVersions(a, b string) int {
	aValid, bValid := IsValidVersion(a), IsValidVersion(b)
	switch {
	case !aValid && !bValid:
		return 0
	case !aValid:
		return -1
	case !bValid:
		return 1
	}

	as := strings.Split(strings.TrimSpace(a), ".")
	bs := strings.Split(strings.TrimSpace(b), ".")
	for i := 0; i < len(as) || i < len(bs); i++ {
		var x, y int
		if i < len(as) {
			x, _ = strconv.Atoi(as[i])
		}
		if i < len(bs) {
			y, _ = strconv.Atoi(bs[i])
		}
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}
