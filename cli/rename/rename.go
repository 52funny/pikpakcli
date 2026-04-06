package rename

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/spf13/cobra"
)

var RenameCmd = &cobra.Command{
	Use:   "rename <path> <new-name>",
	Short: "Rename a file or folder on the PikPak drive",
	Long: `Rename a file or folder on the PikPak drive. 
Example: pikpakcli rename /my-folder/old-name.txt new-name.txt`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		if err := p.Login(); err != nil {
			fmt.Println("Login failed")
			logx.Error(err)
			return nil
		}

		oldPath := args[0]
		newName := strings.TrimSpace(args[1])
		if newName == "" {
			return fmt.Errorf("new name cannot be empty")
		}
		if filepath.Base(newName) != newName {
			return fmt.Errorf("new name must not contain path separators")
		}

		expandedPaths, err := api.ExpandRemotePatterns(&p, "/", []string{oldPath}, false)
		if err != nil {
			fmt.Printf("Could not find file or folder at path '%s'\n", oldPath)
			logx.Error(err)
			return nil
		}
		if len(expandedPaths) != 1 {
			return fmt.Errorf("rename target must match exactly one path")
		}
		oldPath = expandedPaths[0]

		fileStat, err := p.GetFileByPath(oldPath)
		if err != nil {
			fmt.Printf("Could not find file or folder at path '%s'\n", oldPath)
			logx.Error(err)
			return nil
		}

		if err := p.Rename(fileStat.ID, newName); err != nil {
			fmt.Printf("Failed to rename %s\n", oldPath)
			logx.Error(err)
			return nil
		}

		fmt.Printf("Successfully renamed '%s' to '%s'\n", oldPath, newName)
		return nil
	},
}
