package folder

import (
	"fmt"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/spf13/cobra"
)

var NewFolderCommand = &cobra.Command{
	Use:   "folder",
	Short: `Create a folder to pikpak server`,
	Run: func(cmd *cobra.Command, args []string) {
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			fmt.Println("Login failed")
			logx.Error(err)
			return
		}
		if len(args) > 0 {
			handleNewFolder(&p, args)
		} else {
			fmt.Println("Please input the folder name")
		}
	},
}

var path string
var parentId string

func init() {
	NewFolderCommand.Flags().StringVarP(&path, "path", "p", "/", "The path of the folder")
	NewFolderCommand.Flags().StringVarP(&parentId, "parent-id", "P", "", "The parent id")
}

// new folder
func handleNewFolder(p *api.PikPak, folders []string) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			fmt.Println("Get parent id failed")
			logx.Error(err)
			return
		}
	}

	for _, folder := range folders {
		_, err := p.CreateFolder(parentId, folder)
		if err != nil {
			fmt.Printf("Create folder %s failed\n", folder)
			logx.Error(err)
		} else {
			fmt.Printf("Create folder %s success\n", folder)
		}
	}
}
