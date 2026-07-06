package main

import (
	_ "embed"

	"github.com/52funny/pikpakcli/cli"
	"github.com/52funny/pikpakcli/cli/setup"
)

//go:embed config_example.yml
var configTemplate []byte

func main() {
	setup.SetConfigTemplate(configTemplate)
	cli.Execute()
}
