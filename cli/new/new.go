package new

import (
	"github.com/52funny/pikpakcli/cli/new/folder"
	"github.com/52funny/pikpakcli/cli/new/sha"
	"github.com/52funny/pikpakcli/cli/new/url"
	"github.com/spf13/cobra"
)

var NewCommand = &cobra.Command{
	Use:     "new",
	Aliases: []string{"n"},
	Short:   `New can do something like create folder or other things`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func init() {
	NewCommand.AddCommand(folder.NewFolderCommand)
	NewCommand.AddCommand(sha.NewShaCommand)
	NewCommand.AddCommand(url.NewUrlCommand)
}
