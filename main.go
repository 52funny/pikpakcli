package main

import (
	"os"

	"github.com/52funny/pikpakcli/cmd"
	"github.com/52funny/pikpakcli/internal/shell"
)

func main() {
	// Check if any args
	if len(os.Args) == 1 {
		cmd.ExecuteShell(shell.Start)
	} else {
		// If no arg, execute the command directly
		cmd.Execute()
	}
}
