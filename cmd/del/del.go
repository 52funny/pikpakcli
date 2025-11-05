package delete

import (
	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var path string
var err error
var parentId string

var DeleteCmd = &cobra.Command{
	Use:   "delete [filename]",
	Short: "Delete a file or folder on the PikPak server",
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			logrus.Error("Please provide a file name to delete")
			return
		}

		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		if err := p.Login(); err != nil {
			logrus.Fatalf("Login Failed: %v", err)
			return
		}

		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			logrus.Errorln("get path folder id error:", err)
			return
		}

		files, err := p.GetFolderFileStatList(parentId)

		for _, targetName := range args {
			found := false
			for _, f := range files {
				if f.Name == targetName {
					logrus.Debugf("Matched file: Name=%s, ID=%s, Size=%d", f.Name, f.ID, f.Size)
					err = p.DeleteFile(f.ID)
					if err != nil {
						logrus.Errorf("Failed to delete %s: %v", f.Name, err)
					} else {
						logrus.Infof("Deleted: %s", f.Name)
					}
					found = true
					break
				}
			}
			if !found {
				logrus.Errorf("File not found in %s: %s", path, targetName)
			}
		}
	},
}

func init() {
	DeleteCmd.Flags().StringVarP(&path, "path", "p", "/", "The path where to look for the file")
}
