package list

import (
	"fmt"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var long bool
var human bool
var path string
var parentId string

var ListCmd = &cobra.Command{
	Use:   "ls",
	Short: `Get the directory information under the specified folder`,
	Run: func(cmd *cobra.Command, args []string) {
		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			logrus.Errorln("Login Failed:", err)
		}
		long, _ := cmd.Flags().GetBool("long")
		human, _ := cmd.Flags().GetBool("human")
		path, _ := cmd.Flags().GetString("path")
		parentId, _ := cmd.Flags().GetString("parent-id")
		handle(&p, args, long, human, path, parentId)
	},
}

func init() {
	ListCmd.Flags().BoolVarP(&human, "human", "H", false, "display human readable format")
	ListCmd.Flags().BoolVarP(&long, "long", "l", false, "display long format")
	ListCmd.Flags().StringVarP(&path, "path", "p", "/", "display the specified path")
	ListCmd.Flags().StringVarP(&parentId, "parent-id", "P", "", "display the specified parent id")
}

func handle(p *pikpak.PikPak, args []string, long, human bool, path, parentId string) {
	var err error
	if len(args) > 0 {
		path = args[0]
	}
	if parentId == "" {
		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			logrus.Errorln("get path folder id error:", err)
			return
		}
	}
	files, err := p.GetFolderFileStatList(parentId)
	if err != nil {
		logrus.Errorln("get folder file stat list error:", err)
		return
	}
	for _, file := range files {
		if long {
			display(2, &file)
		} else {
			display(0, &file)
		}
	}
}

// mode 0: normal print
// mode 2: long format
func display(mode int, file *pikpak.FileStat) {
	size := utils.FormatStorage(file.Size, human)

	switch mode {
	case 0:
		if file.Kind == "drive#folder" {
			fmt.Printf("%-20s\n", color.GreenString(file.Name))
		} else {
			fmt.Printf("%-20s\n", file.Name)
		}
	case 2:
		if file.Kind == "drive#folder" {
			fmt.Printf("%-26s %-8s %-19s %s\n", file.ID, size, file.CreatedTime.Format("2006-01-02 15:04:05"), color.GreenString(file.Name))
		} else {
			fmt.Printf("%-26s %-8s %-19s %s\n", file.ID, size, file.CreatedTime.Format("2006-01-02 15:04:05"), file.Name)
		}
	}
}
