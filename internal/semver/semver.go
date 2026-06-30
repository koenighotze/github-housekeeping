package semver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type BumpKind int

const (
	Patch BumpKind = iota
	Minor
	Major
)

func (b BumpKind) String() string {
	switch b {
	case Patch:
		return "patch"
	case Minor:
		return "minor"
	case Major:
		return "major"
	default:
		return "unknown"
	}
}

// bumpPattern matches Dependabot PR titles like:
//
//	"Bump <pkg> from <old> to <new>"
//	"chore(deps): bump <pkg> from <old> to <new>"
var bumpPattern = regexp.MustCompile(`(?i)bump\s+\S+\s+from\s+(\S+)\s+to\s+(\S+)`)

// ClassifyBump parses a Dependabot PR title and returns the semver bump kind.
func ClassifyBump(title string) (BumpKind, error) {
	matches := bumpPattern.FindStringSubmatch(title)
	if matches == nil {
		return 0, fmt.Errorf("semver: title does not match Dependabot bump pattern: %q", title)
	}

	oldVer, err := parse(matches[1])
	if err != nil {
		return 0, fmt.Errorf("semver: invalid old version %q: %w", matches[1], err)
	}

	newVer, err := parse(matches[2])
	if err != nil {
		return 0, fmt.Errorf("semver: invalid new version %q: %w", matches[2], err)
	}

	switch {
	case newVer[0] > oldVer[0]:
		return Major, nil
	case newVer[1] > oldVer[1]:
		return Minor, nil
	default:
		return Patch, nil
	}
}

func parse(v string) ([3]int, error) {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)

	var out [3]int
	for i, p := range parts {
		// strip pre-release suffixes like "-beta.1"
		p = strings.SplitN(p, "-", 2)[0]
		n, err := strconv.Atoi(p)
		if err != nil {
			return out, fmt.Errorf("non-numeric segment %q", p)
		}
		out[i] = n
	}

	return out, nil
}
