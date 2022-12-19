package folder

import (
	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var NewFolderCommand = &cobra.Command{
	Use:   "folder",
	Short: `Create a folder to pikpak server`,
	Run: func(cmd *cobra.Command, args []string) {
		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			logrus.Errorln("Login Failed:", err)
		}
		if len(args) > 0 {
			handleNewFolder(&p, args)
		} else {
			logrus.Errorln("Please input the folder name")
		}
	},
}

var path string

func init() {
	NewFolderCommand.Flags().StringVarP(&path, "path", "p", "/", "The path of the folder")
}

// new folder
func handleNewFolder(p *pikpak.PikPak, folders []string) {
	parentId, err := p.GetPathFolderId(path)
	if err != nil {
		logrus.Errorf("Get parent id failed: %s\n", err)
		return
	}

	for _, folder := range folders {
		_, err := p.CreateFolder(parentId, folder)
		if err != nil {
			logrus.Errorf("Create folder %s failed: %s\n", folder, err)
		} else {
			logrus.Infof("Create folder %s success\n", folder)
		}
	}
}
