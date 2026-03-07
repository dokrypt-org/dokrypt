package testrunner

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSuite_ReturnsNonNil(t *testing.T) {
	s := NewSuite("test-suite")
	require.NotNil(t, s)
}

func TestNewSuite_SetsName(t *testing.T) {
	s := NewSuite("my-suite")
	assert.Equal(t, "my-suite", s.Name)
}

func TestNewSuite_EmptyName(t *testing.T) {
	s := NewSuite("")
	assert.Equal(t, "", s.Name)
}

func TestNewSuite_StartsWithNoTests(t *testing.T) {
	s := NewSuite("suite")
	assert.Empty(t, s.Tests)
}

func TestNewSuite_DescriptionDefaultsToEmpty(t *testing.T) {
	s := NewSuite("suite")
	assert.Empty(t, s.Description)
}

func TestNewSuite_SetupDefaultsToNil(t *testing.T) {
	s := NewSuite("suite")
	assert.Nil(t, s.Setup)
}

func TestNewSuite_TeardownDefaultsToNil(t *testing.T) {
	s := NewSuite("suite")
	assert.Nil(t, s.Teardown)
}

func TestAddTest_AppendsSingleTest(t *testing.T) {
	s := NewSuite("suite")
	fn := func(_ context.Context) error { return nil }

	s.AddTest("test-one", fn)

	require.Len(t, s.Tests, 1)
	assert.Equal(t, "test-one", s.Tests[0].Name)
	assert.NotNil(t, s.Tests[0].Fn)
}

func TestAddTest_MultipleTests(t *testing.T) {
	s := NewSuite("suite")
	fn := func(_ context.Context) error { return nil }

	s.AddTest("test-a", fn)
	s.AddTest("test-b", fn)
	s.AddTest("test-c", fn)

	require.Len(t, s.Tests, 3)
	assert.Equal(t, "test-a", s.Tests[0].Name)
	assert.Equal(t, "test-b", s.Tests[1].Name)
	assert.Equal(t, "test-c", s.Tests[2].Name)
}

func TestAddTest_PreservesInsertionOrder(t *testing.T) {
	s := NewSuite("suite")
	fn := func(_ context.Context) error { return nil }

	names := []string{"first", "second", "third", "fourth"}
	for _, name := range names {
		s.AddTest(name, fn)
	}

	for i, name := range names {
		assert.Equal(t, name, s.Tests[i].Name)
	}
}

func TestAddTest_TagsDefaultToNil(t *testing.T) {
	s := NewSuite("suite")
	s.AddTest("test", func(_ context.Context) error { return nil })

	assert.Nil(t, s.Tests[0].Tags)
}

func TestAddTest_FunctionIsCallable(t *testing.T) {
	s := NewSuite("suite")
	called := false
	s.AddTest("test", func(_ context.Context) error {
		called = true
		return nil
	})

	err := s.Tests[0].Fn(context.Background())
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAddTest_DuplicateNamesAllowed(t *testing.T) {
	s := NewSuite("suite")
	fn := func(_ context.Context) error { return nil }

	s.AddTest("same-name", fn)
	s.AddTest("same-name", fn)

	assert.Len(t, s.Tests, 2)
	assert.Equal(t, "same-name", s.Tests[0].Name)
	assert.Equal(t, "same-name", s.Tests[1].Name)
}

func TestAddTaggedTest_AddsTestWithTags(t *testing.T) {
	s := NewSuite("suite")
	fn := func(_ context.Context) error { return nil }

	s.AddTaggedTest("tagged-test", []string{"slow", "integration"}, fn)

	require.Len(t, s.Tests, 1)
	assert.Equal(t, "tagged-test", s.Tests[0].Name)
	assert.Equal(t, []string{"slow", "integration"}, s.Tests[0].Tags)
	assert.NotNil(t, s.Tests[0].Fn)
}

func TestAddTaggedTest_EmptyTags(t *testing.T) {
	s := NewSuite("suite")
	fn := func(_ context.Context) error { return nil }

	s.AddTaggedTest("test", []string{}, fn)

	require.Len(t, s.Tests, 1)
	assert.Empty(t, s.Tests[0].Tags)
}

func TestAddTaggedTest_NilTags(t *testing.T) {
	s := NewSuite("suite")
	fn := func(_ context.Context) error { return nil }

	s.AddTaggedTest("test", nil, fn)

	require.Len(t, s.Tests, 1)
	assert.Nil(t, s.Tests[0].Tags)
}

func TestAddTaggedTest_SingleTag(t *testing.T) {
	s := NewSuite("suite")
	fn := func(_ context.Context) error { return nil }

	s.AddTaggedTest("test", []string{"unit"}, fn)

	require.Len(t, s.Tests[0].Tags, 1)
	assert.Equal(t, "unit", s.Tests[0].Tags[0])
}

func TestAddTaggedTest_FunctionIsCallable(t *testing.T) {
	s := NewSuite("suite")
	called := false

	s.AddTaggedTest("test", []string{"fast"}, func(_ context.Context) error {
		called = true
		return nil
	})

	err := s.Tests[0].Fn(context.Background())
	require.NoError(t, err)
	assert.True(t, called)
}

func TestAddTaggedTest_MultipleTaggedTests(t *testing.T) {
	s := NewSuite("suite")
	fn := func(_ context.Context) error { return nil }

	s.AddTaggedTest("fast-test", []string{"fast"}, fn)
	s.AddTaggedTest("slow-test", []string{"slow", "integration"}, fn)

	require.Len(t, s.Tests, 2)
	assert.Equal(t, []string{"fast"}, s.Tests[0].Tags)
	assert.Equal(t, []string{"slow", "integration"}, s.Tests[1].Tags)
}

func TestSuite_MixedAddTestAndAddTaggedTest(t *testing.T) {
	s := NewSuite("mixed")
	fn := func(_ context.Context) error { return nil }

	s.AddTest("untagged", fn)
	s.AddTaggedTest("tagged", []string{"smoke"}, fn)
	s.AddTest("another-untagged", fn)

	require.Len(t, s.Tests, 3)

	assert.Equal(t, "untagged", s.Tests[0].Name)
	assert.Nil(t, s.Tests[0].Tags)

	assert.Equal(t, "tagged", s.Tests[1].Name)
	assert.Equal(t, []string{"smoke"}, s.Tests[1].Tags)

	assert.Equal(t, "another-untagged", s.Tests[2].Name)
	assert.Nil(t, s.Tests[2].Tags)
}

func TestSuite_SetupCanBeAssigned(t *testing.T) {
	s := NewSuite("suite")
	called := false
	s.Setup = func(_ context.Context) error {
		called = true
		return nil
	}

	err := s.Setup(context.Background())
	require.NoError(t, err)
	assert.True(t, called)
}

func TestSuite_TeardownCanBeAssigned(t *testing.T) {
	s := NewSuite("suite")
	called := false
	s.Teardown = func(_ context.Context) error {
		called = true
		return nil
	}

	err := s.Teardown(context.Background())
	require.NoError(t, err)
	assert.True(t, called)
}

func TestSuite_DescriptionCanBeAssigned(t *testing.T) {
	s := NewSuite("suite")
	s.Description = "Tests for the ERC20 contract"
	assert.Equal(t, "Tests for the ERC20 contract", s.Description)
}

func TestTestCase_FieldsSetCorrectly(t *testing.T) {
	fn := func(_ context.Context) error { return nil }
	tc := TestCase{
		Name: "my-test",
		Fn:   fn,
		Tags: []string{"fast", "unit"},
	}

	assert.Equal(t, "my-test", tc.Name)
	assert.NotNil(t, tc.Fn)
	assert.Equal(t, []string{"fast", "unit"}, tc.Tags)
}

func TestTestCase_ZeroValue(t *testing.T) {
	tc := TestCase{}
	assert.Empty(t, tc.Name)
	assert.Nil(t, tc.Fn)
	assert.Nil(t, tc.Tags)
}
