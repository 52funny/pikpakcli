package list

import (
	"fmt"
	"strconv"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
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
		handle(&p, args)
	},
}

func init() {
	ListCmd.Flags().BoolVarP(&human, "human", "H", false, "display human readable format")
	ListCmd.Flags().BoolVarP(&long, "long", "l", false, "display long format")
	ListCmd.Flags().StringVarP(&path, "path", "p", "/", "display the specified path")
	ListCmd.Flags().StringVarP(&parentId, "parent-id", "P", "", "display the specified parent id")
}

func handle(p *pikpak.PikPak, args []string) {
	var err error
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
			if human {
				display(3, &file)
			} else {
				display(2, &file)
			}
		} else {
			display(0, &file)
		}
	}
}

// lH
// mode 0: normal print
// mode 2: long format
// mode 3: long format and human readable

func display(mode int, file *pikpak.FileStat) {
	switch mode {
	case 0:
		if file.Kind == "drive#folder" {
			fmt.Printf("%6s %-20s\n", file.ID, color.GreenString(file.Name))
		} else {
			fmt.Printf("%6s %-20s\n", file.ID, file.Name)
		}
	case 2:
		if file.Kind == "drive#folder" {
			fmt.Printf("%6s %6s %14s %s\n", file.ID, file.Size, file.CreatedTime.Format("2006-01-02 15:04:05"), color.GreenString(file.Name))
		} else {
			fmt.Printf("%6s %6s %14s %s\n", file.ID, file.Size, file.CreatedTime.Format("2006-01-02 15:04:05"), file.Name)
		}
	case 3:
		if file.Kind == "drive#folder" {
			fmt.Printf("%6s %6s %14s %s\n", file.ID, displayStorage(file.Size), file.CreatedTime.Format("2006-01-02 15:04:05"), color.GreenString(file.Name))
		} else {
			fmt.Printf("%6s %6s %14s %s\n", file.ID, displayStorage(file.Size), file.CreatedTime.Format("2006-01-02 15:04:05"), file.Name)
		}
	}
}

func displayStorage(s string) string {
	size, _ := strconv.ParseUint(s, 10, 64)
	cnt := 0
	for size > 1024 {
		cnt += 1
		if cnt > 5 {
			break
		}
		size /= 1024
	}
	res := strconv.Itoa(int(size))
	switch cnt {
	case 0:
		res += "B"
	case 1:
		res += "KB"
	case 2:
		res += "MB"
	case 3:
		res += "GB"
	case 4:
		res += "TB"
	case 5:
		res += "PB"
	}
	return res
}
