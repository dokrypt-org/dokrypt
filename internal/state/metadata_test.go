package state

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSnapshot_SetsNameAndProject(t *testing.T) {
	opts := SaveOptions{Description: "initial state"}
	snap := NewSnapshot("snap-001", "acme-project", opts)

	require.NotNil(t, snap)
	assert.Equal(t, "snap-001", snap.Name)
	assert.Equal(t, "acme-project", snap.Project)
}

func TestNewSnapshot_SetsDescription(t *testing.T) {
	opts := SaveOptions{Description: "pre-deploy checkpoint"}
	snap := NewSnapshot("s1", "proj", opts)

	assert.Equal(t, "pre-deploy checkpoint", snap.Description)
}

func TestNewSnapshot_SetsTags(t *testing.T) {
	opts := SaveOptions{Tags: []string{"staging", "v2"}}
	snap := NewSnapshot("s1", "proj", opts)

	assert.Equal(t, []string{"staging", "v2"}, snap.Tags)
}

func TestNewSnapshot_SetsCreatedAtNearNow(t *testing.T) {
	before := time.Now().UTC().Add(-time.Second)
	snap := NewSnapshot("s1", "proj", SaveOptions{})
	after := time.Now().UTC().Add(time.Second)

	assert.True(t, snap.CreatedAt.After(before), "CreatedAt should be after start of test")
	assert.True(t, snap.CreatedAt.Before(after), "CreatedAt should be before end of test")
}

func TestNewSnapshot_InitializesEmptyChainsMap(t *testing.T) {
	snap := NewSnapshot("s1", "proj", SaveOptions{})

	require.NotNil(t, snap.Chains)
	assert.Empty(t, snap.Chains)
}

func TestNewSnapshot_NoTagsWhenOptsEmpty(t *testing.T) {
	snap := NewSnapshot("s1", "proj", SaveOptions{})

	assert.Nil(t, snap.Tags)
}

func TestHasTag_ReturnsTrueForExistingTag(t *testing.T) {
	snap := &Snapshot{Tags: []string{"alpha", "beta", "gamma"}}

	assert.True(t, snap.HasTag("alpha"))
	assert.True(t, snap.HasTag("beta"))
	assert.True(t, snap.HasTag("gamma"))
}

func TestHasTag_ReturnsFalseForMissingTag(t *testing.T) {
	snap := &Snapshot{Tags: []string{"alpha", "beta"}}

	assert.False(t, snap.HasTag("delta"))
	assert.False(t, snap.HasTag(""))
}

func TestHasTag_ReturnsFalseOnNilTags(t *testing.T) {
	snap := &Snapshot{}

	assert.False(t, snap.HasTag("anything"))
}

func TestHasTag_CaseSensitive(t *testing.T) {
	snap := &Snapshot{Tags: []string{"Production"}}

	assert.True(t, snap.HasTag("Production"))
	assert.False(t, snap.HasTag("production"))
	assert.False(t, snap.HasTag("PRODUCTION"))
}

func TestAddTag_AppendsNewTag(t *testing.T) {
	snap := &Snapshot{}
	snap.AddTag("v1")

	assert.Equal(t, []string{"v1"}, snap.Tags)
}

func TestAddTag_DoesNotDuplicateExistingTag(t *testing.T) {
	snap := &Snapshot{Tags: []string{"existing"}}
	snap.AddTag("existing")

	assert.Equal(t, []string{"existing"}, snap.Tags)
}

func TestAddTag_MultipleDifferentTags(t *testing.T) {
	snap := &Snapshot{}
	snap.AddTag("a")
	snap.AddTag("b")
	snap.AddTag("c")

	assert.Equal(t, []string{"a", "b", "c"}, snap.Tags)
}

func TestAddTag_DoesNotDuplicateWhenCalledMultipleTimes(t *testing.T) {
	snap := &Snapshot{}
	snap.AddTag("dup")
	snap.AddTag("dup")
	snap.AddTag("dup")

	assert.Len(t, snap.Tags, 1)
}

func TestAddTag_AllowsEmptyString(t *testing.T) {
	snap := &Snapshot{}
	snap.AddTag("")

	assert.Equal(t, []string{""}, snap.Tags)
}

func TestAddTag_EmptyStringIsNotDuplicated(t *testing.T) {
	snap := &Snapshot{}
	snap.AddTag("")
	snap.AddTag("")

	assert.Len(t, snap.Tags, 1)
}

func TestConfigHash_ReturnsDeterministicResult(t *testing.T) {
	data := []byte("project: test\nversion: 1\n")

	h1 := ConfigHash(data)
	h2 := ConfigHash(data)

	assert.Equal(t, h1, h2)
}

func TestConfigHash_DifferentDataProducesDifferentHash(t *testing.T) {
	h1 := ConfigHash([]byte("config-a"))
	h2 := ConfigHash([]byte("config-b"))

	assert.NotEqual(t, h1, h2)
}

func TestConfigHash_ReturnsHexString(t *testing.T) {
	h := ConfigHash([]byte("some config data"))

	for _, ch := range h {
		assert.True(t,
			(ch >= '0' && ch <= '9') || (ch >= 'a' && ch <= 'f'),
			"expected hex character, got %q", ch,
		)
	}
}

func TestConfigHash_ReturnsFirst8BytesOf256BitHash(t *testing.T) {
	h := ConfigHash([]byte("data"))

	assert.Len(t, h, 16, "ConfigHash should return 16 hex characters (8 bytes)")
}

func TestConfigHash_EmptyInput(t *testing.T) {
	h := ConfigHash([]byte{})
	assert.NotEmpty(t, h)
	assert.Len(t, h, 16)
}

func TestConfigHash_SameForEquivalentByteSlices(t *testing.T) {
	data1 := []byte("hello")
	data2 := []byte{'h', 'e', 'l', 'l', 'o'}

	assert.Equal(t, ConfigHash(data1), ConfigHash(data2))
}

func TestAge_ReturnsPositiveDurationForPastTimestamp(t *testing.T) {
	snap := &Snapshot{
		CreatedAt: time.Now().UTC().Add(-5 * time.Minute),
	}

	age := snap.Age()
	assert.True(t, age > 0, "age should be positive for a past timestamp")
}
