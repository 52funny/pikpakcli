package url

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var NewUrlCommand = &cobra.Command{
	Use:   "url",
	Short: `Create a file according to url`,
	Run: func(cmd *cobra.Command, args []string) {
		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			logrus.Errorln("Login Failed:", err)
		}
		if cli {
			handleCli(&p)
			return
		}
		// input mode
		if strings.TrimSpace(input) != "" {
			f, err := os.OpenFile(input, os.O_RDONLY, 0666)
			if err != nil {
				logrus.Errorf("Open file %s failed: %s", input, err)
				return
			}
			reader := bufio.NewReader(f)
			shas := make([]string, 0)
			for {
				lineBytes, _, err := reader.ReadLine()
				if err == io.EOF {
					break
				}
				shas = append(shas, string(lineBytes))
			}
			handleNewUrl(&p, shas)
			return
		}

		// args mode
		if len(args) > 0 {
			handleNewUrl(&p, args)
		} else {
			logrus.Errorln("Please input the folder name")
		}
	},
}

var path string

var parentId string

var name string

var input string

var cli bool

func init() {
	NewUrlCommand.Flags().StringVarP(&path, "path", "p", "/", "The path of the folder")
	NewUrlCommand.Flags().StringVarP(&parentId, "parent-id", "P", "", "The parent id")
	NewUrlCommand.Flags().StringVarP(&name, "name", "n", "", "The name of the task")
	NewUrlCommand.Flags().StringVarP(&input, "input", "i", "", "The input of the sha file")
	NewUrlCommand.Flags().BoolVarP(&cli, "cli", "c", false, "The cli mode")
}

// new folder
func handleNewUrl(p *pikpak.PikPak, shas []string) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			logrus.Errorf("Get parent id failed: %s\n", err)
			return
		}
	}
	for _, url := range shas {
		err := p.CreateUrlFile(parentId, url, name)
		if err != nil {
			logrus.Errorln("Create url file failed: ", err)
			continue
		}
		logrus.Infoln("Create url file success: ", url)
	}
}

func handleCli(p *pikpak.PikPak) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			logrus.Errorf("Get parent id failed: %s\n", err)
			return
		}
	}

	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		lineBytes, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		url := string(lineBytes)
		err = p.CreateUrlFile(parentId, url, name)
		if err != nil {
			logrus.Errorln("Create url file failed: ", err)
			continue
		}
		logrus.Infoln("Create url file success: ", url)
	}
}
