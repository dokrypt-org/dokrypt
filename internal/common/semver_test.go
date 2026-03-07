package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseVersion_ValidBasic(t *testing.T) {
	v, err := ParseVersion("1.2.3")
	require.NoError(t, err)
	assert.Equal(t, 1, v.Major)
	assert.Equal(t, 2, v.Minor)
	assert.Equal(t, 3, v.Patch)
	assert.Empty(t, v.Prerelease)
}

func TestParseVersion_ValidWithVPrefix(t *testing.T) {
	v, err := ParseVersion("v2.0.0")
	require.NoError(t, err)
	assert.Equal(t, 2, v.Major)
	assert.Equal(t, 0, v.Minor)
	assert.Equal(t, 0, v.Patch)
}

func TestParseVersion_ValidZeros(t *testing.T) {
	v, err := ParseVersion("0.0.0")
	require.NoError(t, err)
	assert.Equal(t, 0, v.Major)
	assert.Equal(t, 0, v.Minor)
	assert.Equal(t, 0, v.Patch)
}

func TestParseVersion_ValidPrerelease(t *testing.T) {
	v, err := ParseVersion("1.0.0-alpha.1")
	require.NoError(t, err)
	assert.Equal(t, 1, v.Major)
	assert.Equal(t, 0, v.Minor)
	assert.Equal(t, 0, v.Patch)
	assert.Equal(t, "alpha.1", v.Prerelease)
}

func TestParseVersion_ValidPrereleaseWithVPrefix(t *testing.T) {
	v, err := ParseVersion("v3.1.4-beta")
	require.NoError(t, err)
	assert.Equal(t, 3, v.Major)
	assert.Equal(t, 1, v.Minor)
	assert.Equal(t, 4, v.Patch)
	assert.Equal(t, "beta", v.Prerelease)
}

func TestParseVersion_InvalidMissingPart(t *testing.T) {
	_, err := ParseVersion("1.2")
	assert.Error(t, err)
}

func TestParseVersion_InvalidExtraPart(t *testing.T) {
	_, err := ParseVersion("1.2.3.4")
	assert.Error(t, err)
}

func TestParseVersion_InvalidMajorNotNumeric(t *testing.T) {
	_, err := ParseVersion("x.2.3")
	assert.Error(t, err)
}

func TestParseVersion_InvalidMinorNotNumeric(t *testing.T) {
	_, err := ParseVersion("1.y.3")
	assert.Error(t, err)
}

func TestParseVersion_InvalidPatchNotNumeric(t *testing.T) {
	_, err := ParseVersion("1.2.z")
	assert.Error(t, err)
}

func TestParseVersion_EmptyString(t *testing.T) {
	_, err := ParseVersion("")
	assert.Error(t, err)
}

func TestVersionString_NoPrerelease(t *testing.T) {
	v := Version{Major: 1, Minor: 2, Patch: 3}
	assert.Equal(t, "1.2.3", v.String())
}

func TestVersionString_WithPrerelease(t *testing.T) {
	v := Version{Major: 1, Minor: 0, Patch: 0, Prerelease: "alpha"}
	assert.Equal(t, "1.0.0-alpha", v.String())
}

func TestVersionString_RoundTrip(t *testing.T) {
	original := "2.5.11-rc.3"
	v, err := ParseVersion(original)
	require.NoError(t, err)
	assert.Equal(t, original, v.String())
}

func TestLessThan_MajorDifference(t *testing.T) {
	a := Version{Major: 1}
	b := Version{Major: 2}
	assert.True(t, a.LessThan(b))
	assert.False(t, b.LessThan(a))
}

func TestLessThan_MinorDifference(t *testing.T) {
	a := Version{Major: 1, Minor: 1}
	b := Version{Major: 1, Minor: 2}
	assert.True(t, a.LessThan(b))
	assert.False(t, b.LessThan(a))
}

func TestLessThan_PatchDifference(t *testing.T) {
	a := Version{Major: 1, Minor: 2, Patch: 3}
	b := Version{Major: 1, Minor: 2, Patch: 4}
	assert.True(t, a.LessThan(b))
	assert.False(t, b.LessThan(a))
}

func TestLessThan_Equal(t *testing.T) {
	a := Version{Major: 1, Minor: 2, Patch: 3}
	b := Version{Major: 1, Minor: 2, Patch: 3}
	assert.False(t, a.LessThan(b))
	assert.False(t, b.LessThan(a))
}

func TestLessThan_PrereleaseIsLower(t *testing.T) {
	pre := Version{Major: 1, Prerelease: "alpha"}
	release := Version{Major: 1}
	assert.True(t, pre.LessThan(release))
	assert.False(t, release.LessThan(pre))
}

func TestLessThan_PrereleaseLexicographic(t *testing.T) {
	a := Version{Major: 1, Prerelease: "alpha"}
	b := Version{Major: 1, Prerelease: "beta"}
	assert.True(t, a.LessThan(b))
	assert.False(t, b.LessThan(a))
}

func TestLessThan_TwoPrereleaseEqual(t *testing.T) {
	a := Version{Major: 1, Prerelease: "rc.1"}
	b := Version{Major: 1, Prerelease: "rc.1"}
	assert.False(t, a.LessThan(b))
	assert.False(t, b.LessThan(a))
}

func TestEqual_Identical(t *testing.T) {
	a := Version{Major: 1, Minor: 2, Patch: 3}
	b := Version{Major: 1, Minor: 2, Patch: 3}
	assert.True(t, a.Equal(b))
}

func TestEqual_DifferentPatch(t *testing.T) {
	a := Version{Major: 1, Minor: 2, Patch: 3}
	b := Version{Major: 1, Minor: 2, Patch: 4}
	assert.False(t, a.Equal(b))
}

func TestEqual_PrereleaseMatters(t *testing.T) {
	a := Version{Major: 1, Prerelease: "alpha"}
	b := Version{Major: 1}
	assert.False(t, a.Equal(b))
}

func TestEqual_SamePrerelease(t *testing.T) {
	a := Version{Major: 1, Prerelease: "rc.1"}
	b := Version{Major: 1, Prerelease: "rc.1"}
	assert.True(t, a.Equal(b))
}

func TestParseConstraint_Exact(t *testing.T) {
	c, err := ParseConstraint("1.2.3")
	require.NoError(t, err)
	assert.Equal(t, ConstraintExact, c.Type)
	assert.Equal(t, Version{Major: 1, Minor: 2, Patch: 3}, c.Version)
}

func TestParseConstraint_ExactWithEquals(t *testing.T) {
	c, err := ParseConstraint("=1.2.3")
	require.NoError(t, err)
	assert.Equal(t, ConstraintExact, c.Type)
}

func TestParseConstraint_Caret(t *testing.T) {
	c, err := ParseConstraint("^1.2.3")
	require.NoError(t, err)
	assert.Equal(t, ConstraintCaret, c.Type)
	assert.Equal(t, Version{Major: 1, Minor: 2, Patch: 3}, c.Version)
}

func TestParseConstraint_Tilde(t *testing.T) {
	c, err := ParseConstraint("~1.2.3")
	require.NoError(t, err)
	assert.Equal(t, ConstraintTilde, c.Type)
}

func TestParseConstraint_GTE(t *testing.T) {
	c, err := ParseConstraint(">=1.0.0")
	require.NoError(t, err)
	assert.Equal(t, ConstraintGTE, c.Type)
}

func TestParseConstraint_GT(t *testing.T) {
	c, err := ParseConstraint(">1.0.0")
	require.NoError(t, err)
	assert.Equal(t, ConstraintGT, c.Type)
}

func TestParseConstraint_LTE(t *testing.T) {
	c, err := ParseConstraint("<=2.0.0")
	require.NoError(t, err)
	assert.Equal(t, ConstraintLTE, c.Type)
}

func TestParseConstraint_LT(t *testing.T) {
	c, err := ParseConstraint("<2.0.0")
	require.NoError(t, err)
	assert.Equal(t, ConstraintLT, c.Type)
}

func TestParseConstraint_InvalidVersion(t *testing.T) {
	_, err := ParseConstraint("^notaversion")
	assert.Error(t, err)
}

func TestParseConstraint_WhitespaceStripped(t *testing.T) {
	c, err := ParseConstraint("  ^1.0.0  ")
	require.NoError(t, err)
	assert.Equal(t, ConstraintCaret, c.Type)
}

func TestMatchesExact_Equal(t *testing.T) {
	c, _ := ParseConstraint("1.2.3")
	v, _ := ParseVersion("1.2.3")
	assert.True(t, c.Matches(v))
}

func TestMatchesExact_Different(t *testing.T) {
	c, _ := ParseConstraint("1.2.3")
	v, _ := ParseVersion("1.2.4")
	assert.False(t, c.Matches(v))
}

func TestMatchesCaret_SameVersion(t *testing.T) {
	c, _ := ParseConstraint("^1.2.3")
	v, _ := ParseVersion("1.2.3")
	assert.True(t, c.Matches(v))
}

func TestMatchesCaret_HigherPatch(t *testing.T) {
	c, _ := ParseConstraint("^1.2.3")
	v, _ := ParseVersion("1.2.9")
	assert.True(t, c.Matches(v))
}

func TestMatchesCaret_HigherMinor(t *testing.T) {
	c, _ := ParseConstraint("^1.2.3")
	v, _ := ParseVersion("1.9.0")
	assert.True(t, c.Matches(v))
}

func TestMatchesCaret_NextMajor(t *testing.T) {
	c, _ := ParseConstraint("^1.2.3")
	v, _ := ParseVersion("2.0.0")
	assert.False(t, c.Matches(v))
}

func TestMatchesCaret_LowerPatch(t *testing.T) {
	c, _ := ParseConstraint("^1.2.3")
	v, _ := ParseVersion("1.2.2")
	assert.False(t, c.Matches(v))
}

func TestMatchesCaret_LowerMajor(t *testing.T) {
	c, _ := ParseConstraint("^2.0.0")
	v, _ := ParseVersion("1.9.9")
	assert.False(t, c.Matches(v))
}

func TestMatchesTilde_SameVersion(t *testing.T) {
	c, _ := ParseConstraint("~1.2.3")
	v, _ := ParseVersion("1.2.3")
	assert.True(t, c.Matches(v))
}

func TestMatchesTilde_HigherPatch(t *testing.T) {
	c, _ := ParseConstraint("~1.2.3")
	v, _ := ParseVersion("1.2.9")
	assert.True(t, c.Matches(v))
}

func TestMatchesTilde_NextMinor(t *testing.T) {
	c, _ := ParseConstraint("~1.2.3")
	v, _ := ParseVersion("1.3.0")
	assert.False(t, c.Matches(v))
}

func TestMatchesTilde_LowerPatch(t *testing.T) {
	c, _ := ParseConstraint("~1.2.3")
	v, _ := ParseVersion("1.2.2")
	assert.False(t, c.Matches(v))
}

func TestMatchesGTE_Equal(t *testing.T) {
	c, _ := ParseConstraint(">=1.0.0")
	v, _ := ParseVersion("1.0.0")
	assert.True(t, c.Matches(v))
}

func TestMatchesGTE_Greater(t *testing.T) {
	c, _ := ParseConstraint(">=1.0.0")
	v, _ := ParseVersion("2.0.0")
	assert.True(t, c.Matches(v))
}

func TestMatchesGTE_Less(t *testing.T) {
	c, _ := ParseConstraint(">=1.0.0")
	v, _ := ParseVersion("0.9.9")
	assert.False(t, c.Matches(v))
}

func TestMatchesGT_Greater(t *testing.T) {
	c, _ := ParseConstraint(">1.0.0")
	v, _ := ParseVersion("1.0.1")
	assert.True(t, c.Matches(v))
}

func TestMatchesGT_Equal(t *testing.T) {
	c, _ := ParseConstraint(">1.0.0")
	v, _ := ParseVersion("1.0.0")
	assert.False(t, c.Matches(v))
}

func TestMatchesGT_Less(t *testing.T) {
	c, _ := ParseConstraint(">1.0.0")
	v, _ := ParseVersion("0.9.0")
	assert.False(t, c.Matches(v))
}

func TestMatchesLTE_Equal(t *testing.T) {
	c, _ := ParseConstraint("<=2.0.0")
	v, _ := ParseVersion("2.0.0")
	assert.True(t, c.Matches(v))
}

func TestMatchesLTE_Less(t *testing.T) {
	c, _ := ParseConstraint("<=2.0.0")
	v, _ := ParseVersion("1.9.9")
	assert.True(t, c.Matches(v))
}

func TestMatchesLTE_Greater(t *testing.T) {
	c, _ := ParseConstraint("<=2.0.0")
	v, _ := ParseVersion("2.0.1")
	assert.False(t, c.Matches(v))
}

func TestMatchesLT_Less(t *testing.T) {
	c, _ := ParseConstraint("<2.0.0")
	v, _ := ParseVersion("1.9.9")
	assert.True(t, c.Matches(v))
}

func TestMatchesLT_Equal(t *testing.T) {
	c, _ := ParseConstraint("<2.0.0")
	v, _ := ParseVersion("2.0.0")
	assert.False(t, c.Matches(v))
}

func TestMatchesLT_Greater(t *testing.T) {
	c, _ := ParseConstraint("<2.0.0")
	v, _ := ParseVersion("2.0.1")
	assert.False(t, c.Matches(v))
}
