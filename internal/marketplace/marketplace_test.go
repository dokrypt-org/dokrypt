package marketplace

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestPackageMeta_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	meta := PackageMeta{
		Name:        "my-template",
		Version:     "1.0.0",
		Description: "A test template",
		Author:      "tester",
		Category:    "defi",
		Difficulty:  "beginner",
		Tags:        []string{"solidity", "evm"},
		Chains:      []string{"ethereum", "polygon"},
		Services:    []string{"hardhat"},
		License:     "MIT",
		Premium:     false,
		Price:       "",
		Downloads:   42,
		Stars:       5,
		Homepage:    "https://example.com",
		Repository:  "https://github.com/example/repo",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := json.Marshal(meta)
	require.NoError(t, err)

	var decoded PackageMeta
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, meta.Name, decoded.Name)
	assert.Equal(t, meta.Version, decoded.Version)
	assert.Equal(t, meta.Description, decoded.Description)
	assert.Equal(t, meta.Author, decoded.Author)
	assert.Equal(t, meta.Category, decoded.Category)
	assert.Equal(t, meta.Difficulty, decoded.Difficulty)
	assert.Equal(t, meta.Tags, decoded.Tags)
	assert.Equal(t, meta.Chains, decoded.Chains)
	assert.Equal(t, meta.Services, decoded.Services)
	assert.Equal(t, meta.License, decoded.License)
	assert.Equal(t, meta.Premium, decoded.Premium)
	assert.Equal(t, meta.Price, decoded.Price)
	assert.Equal(t, meta.Downloads, decoded.Downloads)
	assert.Equal(t, meta.Stars, decoded.Stars)
	assert.Equal(t, meta.Homepage, decoded.Homepage)
	assert.Equal(t, meta.Repository, decoded.Repository)
}

func TestPackageMeta_JSONOmitsEmptyOptionalFields(t *testing.T) {
	meta := PackageMeta{
		Name:    "minimal",
		Version: "0.1.0",
	}

	data, err := json.Marshal(meta)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	_, hasHomepage := raw["homepage"]
	_, hasRepository := raw["repository"]
	assert.False(t, hasHomepage, "empty homepage should be omitted from JSON")
	assert.False(t, hasRepository, "empty repository should be omitted from JSON")
}

func TestPackageMeta_YAMLRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	meta := PackageMeta{
		Name:        "yaml-template",
		Version:     "2.0.0",
		Description: "A YAML test template",
		Author:      "yaml-author",
		Category:    "nft",
		Difficulty:  "advanced",
		Tags:        []string{"rust", "solana"},
		Chains:      []string{"solana"},
		Services:    []string{"anchor"},
		License:     "Apache-2.0",
		Premium:     true,
		Price:       "$9.99",
		Downloads:   100,
		Stars:       20,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	data, err := yaml.Marshal(meta)
	require.NoError(t, err)

	var decoded PackageMeta
	err = yaml.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, meta.Name, decoded.Name)
	assert.Equal(t, meta.Version, decoded.Version)
	assert.Equal(t, meta.Description, decoded.Description)
	assert.Equal(t, meta.Author, decoded.Author)
	assert.Equal(t, meta.Category, decoded.Category)
	assert.Equal(t, meta.Premium, decoded.Premium)
	assert.Equal(t, meta.Price, decoded.Price)
	assert.Equal(t, meta.Tags, decoded.Tags)
	assert.Equal(t, meta.Chains, decoded.Chains)
}

func TestSearchResult_JSONRoundTrip(t *testing.T) {
	result := SearchResult{
		Query: "defi",
		Total: 2,
		Packages: []PackageMeta{
			{Name: "pkg1", Version: "1.0.0"},
			{Name: "pkg2", Version: "2.0.0"},
		},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded SearchResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, result.Query, decoded.Query)
	assert.Equal(t, result.Total, decoded.Total)
	assert.Len(t, decoded.Packages, 2)
	assert.Equal(t, "pkg1", decoded.Packages[0].Name)
	assert.Equal(t, "pkg2", decoded.Packages[1].Name)
}

func TestSearchResult_EmptyPackages(t *testing.T) {
	result := SearchResult{
		Query:    "nonexistent",
		Total:    0,
		Packages: []PackageMeta{},
	}

	data, err := json.Marshal(result)
	require.NoError(t, err)

	var decoded SearchResult
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 0, decoded.Total)
	assert.Empty(t, decoded.Packages)
}

func TestInstalledPackage_JSONRoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	installed := InstalledPackage{
		PackageMeta: PackageMeta{
			Name:    "installed-pkg",
			Version: "1.2.3",
			Author:  "dev",
		},
		InstalledAt: now,
		Path:        "/home/user/.dokrypt/marketplace/installed-pkg",
	}

	data, err := json.Marshal(installed)
	require.NoError(t, err)

	var decoded InstalledPackage
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, installed.Name, decoded.Name)
	assert.Equal(t, installed.Version, decoded.Version)
	assert.Equal(t, installed.Author, decoded.Author)
	assert.Equal(t, installed.Path, decoded.Path)
}

func TestInstalledPackage_InlineEmbed(t *testing.T) {
	installed := InstalledPackage{
		PackageMeta: PackageMeta{
			Name:    "inline-test",
			Version: "0.0.1",
		},
		Path: "/tmp/test",
	}

	data, err := json.Marshal(installed)
	require.NoError(t, err)

	var raw map[string]interface{}
	err = json.Unmarshal(data, &raw)
	require.NoError(t, err)

	assert.Equal(t, "inline-test", raw["name"])
	assert.Equal(t, "0.0.1", raw["version"])
	assert.Equal(t, "/tmp/test", raw["path"])
}

func TestPackageMeta_ZeroValue(t *testing.T) {
	var meta PackageMeta
	assert.Empty(t, meta.Name)
	assert.Empty(t, meta.Version)
	assert.False(t, meta.Premium)
	assert.Equal(t, 0, meta.Downloads)
	assert.Equal(t, 0, meta.Stars)
	assert.Nil(t, meta.Tags)
	assert.Nil(t, meta.Chains)
	assert.Nil(t, meta.Services)
	assert.True(t, meta.CreatedAt.IsZero())
	assert.True(t, meta.UpdatedAt.IsZero())
}
