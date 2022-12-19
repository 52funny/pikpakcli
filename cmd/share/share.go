package share

import (
	"fmt"
	"os"
	"path/filepath"
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

func init() {
	ShareCommand.Flags().StringVarP(&folder, "path", "p", "", "specific the folder of the pikpak server")
	ShareCommand.Flags().StringVarP(&output, "output", "o", "", "specific the file to write")
}

// Share folder
func shareFolder(p *pikpak.PikPak, f *os.File) {
	path := filepath.Join(folder)
	dirs := strings.Split(path, "/")
	parentId, err := p.GetDeepParentId("", dirs)
	if err != nil {
		logrus.Errorln("Get parent id failed:", err)
		return
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
	files := make([]string, 0, len(args))
	for _, v := range args {
		files = append(files, filepath.Join(folder, v))
	}
	for _, file := range files {
		dir, base := filepath.Dir(file), filepath.Base(file)
		dirSplit := strings.Split(dir, "/")
		id, err := p.GetDeepParentId("", dirSplit)
		if err != nil {
			logrus.Errorln(dir, "Get Parent Folder Id Failed:", err)
			continue
		}
		stat, err := p.GetFileStat(id, base)
		if err != nil {
			logrus.Errorln(dir, "Get File Stat Failed:", err)
			continue
		}
		fmt.Fprintf(f, "PikPak://%s|%s|%s\n", stat.Name, stat.Size, stat.Hash)
	}
}
