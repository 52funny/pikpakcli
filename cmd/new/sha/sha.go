package sha

import (
	"bufio"
	"io"
	"os"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var NewShaCommand = &cobra.Command{
	Use:   "sha",
	Short: `Create a file according to sha`,
	Run: func(cmd *cobra.Command, args []string) {
		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			logrus.Errorln("Login Failed:", err)
		}
		// input mode
		if strings.TrimSpace(input) != "" {
			f, err := os.OpenFile(input, os.O_RDONLY, 0666)
			if err != nil {
				logrus.Errorln("Open file %s failed:", input, err)
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
			handleNewSha(&p, shas)
			return
		}

		// args mode
		if len(args) > 0 {
			handleNewSha(&p, args)
		} else {
			logrus.Errorln("Please input the folder name")
		}
	},
}

var path string

var parentId string

var input string

func init() {
	NewShaCommand.Flags().StringVarP(&path, "path", "p", "/", "The path of the folder")
	NewShaCommand.Flags().StringVarP(&input, "input", "i", "", "The input of the sha file")
	NewShaCommand.Flags().StringVarP(&parentId, "parent-id", "P", "", "The parent id")
}

// new folder
func handleNewSha(p *pikpak.PikPak, shas []string) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			logrus.Errorf("Get parent id failed: %s\n", err)
			return
		}
	}

	for _, sha := range shas {
		sha = sha[strings.Index(sha, "://")+3:]
		shaElements := strings.Split(sha, "|")
		if len(shaElements) != 3 {
			logrus.Errorln("The sha format is wrong: ", sha)
			continue
		}
		name, size, sha := shaElements[0], shaElements[1], shaElements[2]
		err := p.CreateShaFile(parentId, name, size, sha)
		if err != nil {
			logrus.Errorln("Create sha file failed: ", err)
			continue
		}
		logrus.Infoln("Create sha file success: ", name)
	}
}
