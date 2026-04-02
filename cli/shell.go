package cli

import (
	ishell "github.com/52funny/pikpakcli/internal/shell"
	"github.com/spf13/cobra"
)

var shellCmd = &cobra.Command{
	Use:   "shell",
	Short: "Start an interactive PikPak shell",
	Run: func(cmd *cobra.Command, args []string) {
		ishell.Start(rootCmd)
	},
}
