package rename

import (
	"fmt"
	"path/filepath"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var RenameCmd = &cobra.Command{
	Use:   "rename <path> <new-name>",
	Short: "Rename a file or folder on the PikPak drive",
	Long:  `Rename a file or folder on the PikPak drive. 
Example: pikpakcli rename /my-folder/old-name.txt new-name.txt`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		if err := p.Login(); err != nil {
			logrus.Errorln("Login Failed:", err)
			return err
		}

		oldPath := args[0]
		newName := filepath.Base(args[1])

		fileStat, err := p.GetFileByPath(oldPath)
		if err != nil {
			logrus.Errorf("Could not find file or folder at path '%s': %v", oldPath, err)
			return err
		}

		if err := p.Rename(fileStat.ID, newName); err != nil {
			logrus.Errorf("Failed to rename file: %v", err)
			return err
		}

		fmt.Printf("Successfully renamed '%s' to '%s'\n", oldPath, newName)
		return nil
	},
}
