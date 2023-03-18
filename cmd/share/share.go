package share

import (
	"fmt"
	"os"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var ShareCommand = &cobra.Command{
	Use:     "share",
	Aliases: []string{"d"},
	Short:   `Share file links on the pikpak server`,
	Run: func(cmd *cobra.Command, args []string) {
		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			logrus.Errorln("Login Failed:", err)
		}
		// Output file handle
		var f = os.Stdout
		if strings.TrimSpace(output) != "" {
			file, err := os.Create(output)
			if err != nil {
				logrus.Errorln("Create file failed:", err)
				return
			}
			defer file.Close()
			f = file
		}

		if len(args) > 0 {
			shareFiles(&p, args, f)
		} else {
			shareFolder(&p, f)
		}
	},
}

// Specifies the folder of the pikpak server
// default is the root folder
var folder string

// Specifies the file to write
// default is the stdout
var output string

var parentId string

func init() {
	ShareCommand.Flags().StringVarP(&folder, "path", "p", "/", "specific the folder of the pikpak server")
	ShareCommand.Flags().StringVarP(&output, "output", "o", "", "specific the file to write")
	ShareCommand.Flags().StringVarP(&parentId, "parent-id", "P", "", "parent folder id")
}

// Share folder
func shareFolder(p *pikpak.PikPak, f *os.File) {
	var err error
	if parentId == "" {
		parentId, err = p.GetDeepFolderId("", folder)
		if err != nil {
			logrus.Errorln("Get parent id failed:", err)
			return
		}
	}
	fileStat, err := p.GetFolderFileStatList(parentId)
	if err != nil {
		logrus.Errorln("Get folder file stat list failed:", err)
		return
	}
	for _, stat := range fileStat {
		// logrus.Debug(stat)
		if stat.Kind == "drive#file" {
			fmt.Fprintf(f, "PikPak://%s|%s|%s\n", stat.Name, stat.Size, stat.Hash)
		}
	}
}

// Share files
func shareFiles(p *pikpak.PikPak, args []string, f *os.File) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(folder)
		if err != nil {
			logrus.Errorln("get parent id failed:", err)
			return
		}
	}
	for _, path := range args {
		stat, err := p.GetFileStat(parentId, path)
		if err != nil {
			logrus.Errorln(path, "get file stat error:", err)
			continue
		}
		fmt.Fprintf(f, "PikPak://%s|%s|%s\n", stat.Name, stat.Size, stat.Hash)
	}
}
