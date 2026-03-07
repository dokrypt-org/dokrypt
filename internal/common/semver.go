package common

import (
	"fmt"
	"strconv"
	"strings"
)

type Version struct {
	Major      int
	Minor      int
	Patch      int
	Prerelease string
}

func ParseVersion(s string) (Version, error) {
	s = strings.TrimPrefix(s, "v")

	var prerelease string
	if idx := strings.Index(s, "-"); idx != -1 {
		prerelease = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("invalid semver: expected 3 parts, got %d", len(parts))
	}

	major, err := strconv.Atoi(parts[0])
	if err != nil {
		return Version{}, fmt.Errorf("invalid major version: %w", err)
	}

	minor, err := strconv.Atoi(parts[1])
	if err != nil {
		return Version{}, fmt.Errorf("invalid minor version: %w", err)
	}

	patch, err := strconv.Atoi(parts[2])
	if err != nil {
		return Version{}, fmt.Errorf("invalid patch version: %w", err)
	}

	return Version{
		Major:      major,
		Minor:      minor,
		Patch:      patch,
		Prerelease: prerelease,
	}, nil
}

func (v Version) String() string {
	s := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.Prerelease != "" {
		s += "-" + v.Prerelease
	}
	return s
}

func (v Version) LessThan(other Version) bool {
	if v.Major != other.Major {
		return v.Major < other.Major
	}
	if v.Minor != other.Minor {
		return v.Minor < other.Minor
	}
	if v.Patch != other.Patch {
		return v.Patch < other.Patch
	}
	if v.Prerelease == "" && other.Prerelease != "" {
		return false
	}
	if v.Prerelease != "" && other.Prerelease == "" {
		return true
	}
	return v.Prerelease < other.Prerelease
}

func (v Version) Equal(other Version) bool {
	return v.Major == other.Major &&
		v.Minor == other.Minor &&
		v.Patch == other.Patch &&
		v.Prerelease == other.Prerelease
}

type ConstraintType int

const (
	ConstraintExact  ConstraintType = iota // =1.2.3
	ConstraintCaret                        // ^1.2.3 (>=1.2.3, <2.0.0)
	ConstraintTilde                        // ~1.2.3 (>=1.2.3, <1.3.0)
	ConstraintGTE                          // >=1.2.3
	ConstraintGT                           // >1.2.3
	ConstraintLTE                          // <=1.2.3
	ConstraintLT                           // <1.2.3
)

type Constraint struct {
	Type    ConstraintType
	Version Version
}

func ParseConstraint(s string) (Constraint, error) {
	s = strings.TrimSpace(s)

	var ctype ConstraintType
	var versionStr string

	switch {
	case strings.HasPrefix(s, "^"):
		ctype = ConstraintCaret
		versionStr = s[1:]
	case strings.HasPrefix(s, "~"):
		ctype = ConstraintTilde
		versionStr = s[1:]
	case strings.HasPrefix(s, ">="):
		ctype = ConstraintGTE
		versionStr = s[2:]
	case strings.HasPrefix(s, ">"):
		ctype = ConstraintGT
		versionStr = s[1:]
	case strings.HasPrefix(s, "<="):
		ctype = ConstraintLTE
		versionStr = s[2:]
	case strings.HasPrefix(s, "<"):
		ctype = ConstraintLT
		versionStr = s[1:]
	case strings.HasPrefix(s, "="):
		ctype = ConstraintExact
		versionStr = s[1:]
	default:
		ctype = ConstraintExact
		versionStr = s
	}

	v, err := ParseVersion(strings.TrimSpace(versionStr))
	if err != nil {
		return Constraint{}, fmt.Errorf("invalid constraint %q: %w", s, err)
	}

	return Constraint{Type: ctype, Version: v}, nil
}

func (c Constraint) Matches(v Version) bool {
	switch c.Type {
	case ConstraintExact:
		return v.Equal(c.Version)
	case ConstraintCaret:
		if v.LessThan(c.Version) {
			return false
		}
		upper := Version{Major: c.Version.Major + 1}
		return v.LessThan(upper)
	case ConstraintTilde:
		if v.LessThan(c.Version) {
			return false
		}
		upper := Version{Major: c.Version.Major, Minor: c.Version.Minor + 1}
		return v.LessThan(upper)
	case ConstraintGTE:
		return !v.LessThan(c.Version)
	case ConstraintGT:
		return !v.LessThan(c.Version) && !v.Equal(c.Version)
	case ConstraintLTE:
		return v.LessThan(c.Version) || v.Equal(c.Version)
	case ConstraintLT:
		return v.LessThan(c.Version)
	}
	return false
}
