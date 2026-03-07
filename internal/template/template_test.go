package template

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplateStruct(t *testing.T) {
	tmpl := Template{
		Name:        "test-template",
		Version:     "1.0.0",
		Description: "A test template",
		Author:      "tester",
		Tags:        []string{"test", "example"},
		Chains:      []string{"ethereum"},
		Services:    []string{"ipfs"},
		License:     "MIT",
		Category:    "basic",
		Difficulty:  "beginner",
		Premium:     false,
		Price:       "free",
	}

	assert.Equal(t, "test-template", tmpl.Name)
	assert.Equal(t, "1.0.0", tmpl.Version)
	assert.Equal(t, "A test template", tmpl.Description)
	assert.Equal(t, "tester", tmpl.Author)
	assert.Equal(t, []string{"test", "example"}, tmpl.Tags)
	assert.Equal(t, []string{"ethereum"}, tmpl.Chains)
	assert.Equal(t, []string{"ipfs"}, tmpl.Services)
	assert.Equal(t, "MIT", tmpl.License)
	assert.Equal(t, "basic", tmpl.Category)
	assert.Equal(t, "beginner", tmpl.Difficulty)
	assert.False(t, tmpl.Premium)
	assert.Equal(t, "free", tmpl.Price)
}

func TestTemplateZeroValue(t *testing.T) {
	var tmpl Template
	assert.Empty(t, tmpl.Name)
	assert.Empty(t, tmpl.Version)
	assert.Empty(t, tmpl.Description)
	assert.Empty(t, tmpl.Author)
	assert.Nil(t, tmpl.Tags)
	assert.Nil(t, tmpl.Chains)
	assert.Nil(t, tmpl.Services)
	assert.Empty(t, tmpl.License)
	assert.Empty(t, tmpl.Category)
	assert.Empty(t, tmpl.Difficulty)
	assert.False(t, tmpl.Premium)
	assert.Empty(t, tmpl.Price)
}

func TestTemplatePremiumFlag(t *testing.T) {
	free := Template{Premium: false, Price: "free"}
	premium := Template{Premium: true, Price: "$49"}

	assert.False(t, free.Premium)
	assert.Equal(t, "free", free.Price)
	assert.True(t, premium.Premium)
	assert.Equal(t, "$49", premium.Price)
}

func TestVarsStruct(t *testing.T) {
	vars := Vars{
		ProjectName: "my-project",
		ChainName:   "ethereum",
		ChainID:     1,
		Engine:      "hardhat",
		Author:      "dev",
	}

	assert.Equal(t, "my-project", vars.ProjectName)
	assert.Equal(t, "ethereum", vars.ChainName)
	assert.Equal(t, uint64(1), vars.ChainID)
	assert.Equal(t, "hardhat", vars.Engine)
	assert.Equal(t, "dev", vars.Author)
}

func TestVarsZeroValue(t *testing.T) {
	var vars Vars
	assert.Empty(t, vars.ProjectName)
	assert.Empty(t, vars.ChainName)
	assert.Equal(t, uint64(0), vars.ChainID)
	assert.Empty(t, vars.Engine)
	assert.Empty(t, vars.Author)
}

func TestInfoStruct(t *testing.T) {
	info := Info{
		Template: Template{
			Name:    "evm-basic",
			Version: "2.0.0",
		},
		Path:    "/path/to/template",
		BuiltIn: true,
	}

	assert.Equal(t, "evm-basic", info.Template.Name)
	assert.Equal(t, "2.0.0", info.Template.Version)
	assert.Equal(t, "/path/to/template", info.Path)
	assert.True(t, info.BuiltIn)
}

func TestInfoZeroValue(t *testing.T) {
	var info Info
	assert.Empty(t, info.Template.Name)
	assert.Empty(t, info.Path)
	assert.False(t, info.BuiltIn)
}

func TestInfoBuiltInVsExternal(t *testing.T) {
	builtIn := Info{
		Template: Template{Name: "evm-basic"},
		BuiltIn:  true,
	}
	external := Info{
		Template: Template{Name: "custom"},
		Path:     "/home/user/.dokrypt/templates/custom",
		BuiltIn:  false,
	}

	assert.True(t, builtIn.BuiltIn)
	assert.Empty(t, builtIn.Path)

	assert.False(t, external.BuiltIn)
	assert.NotEmpty(t, external.Path)
}
