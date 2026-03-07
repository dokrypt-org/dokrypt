package builtin

import (
	"embed"
	"io/fs"
)

//go:embed all:templates
var templatesFS embed.FS

func FS() fs.FS {
	sub, err := fs.Sub(templatesFS, "templates")
	if err != nil {
		panic("builtin: templates directory not found in embed: " + err.Error())
	}
	return sub
}

func TemplateFS(name string) (fs.FS, error) {
	return fs.Sub(FS(), name)
}

func Names() []string {
	return []string{
		"evm-basic",
		"evm-defi",
		"evm-nft",
		"evm-dao",
		"evm-token",
	}
}
