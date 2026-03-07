package main

import (
	"os"

	"github.com/dokrypt/dokrypt/internal/cli"
)

func main() {
	os.Exit(cli.Execute())
}
