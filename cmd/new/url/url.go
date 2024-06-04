package url

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/spf13/cobra"
)

var (
	path, parentId, input, outputFormatVal string
	cli                                    bool
	err                                    error
	logClient                              utils.Log
)

var NewUrlCommand = &cobra.Command{
	Use:   "url",
	Short: `Create a file according to url`,
	Run: func(cmd *cobra.Command, args []string) {

		flagset := cmd.InheritedFlags()
		outputFormatVal, err = flagset.GetString("output")
		if err != nil {
			panic(err)
		}

		logClient = utils.NewLog(outputFormatVal)

		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		err = p.Login()
		if err != nil {
			logClient.Errorf("Login Failed: %v", err)
		}
		if cli {
			handleCli(&p)
			return
		}
		// input mode
		if strings.TrimSpace(input) != "" {
			f, err := os.OpenFile(input, os.O_RDONLY, 0666)
			if err != nil {
				logClient.Errorf("Open file %s failed: %v", input, err)
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
			logClient.Error("Please input the folder name")
		}
	},
}

func init() {
	NewUrlCommand.Flags().StringVarP(&path, "path", "p", "/", "The path of the folder")
	NewUrlCommand.Flags().StringVarP(&parentId, "parent-id", "P", "", "The parent id")
	NewUrlCommand.Flags().StringVarP(&input, "input", "i", "", "The input of the sha file")
	NewUrlCommand.Flags().BoolVarP(&cli, "cli", "c", false, "The cli mode")
}

// new folder
func handleNewUrl(p *pikpak.PikPak, shas []string) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			logClient.Errorf("Get parent id failed: %v\n", err)
			return
		}
	}
	for _, url := range shas {
		taskInfo, err := p.CreateUrlFile(parentId, url)
		if err != nil {
			logClient.Errorf("Create url file failed: %v\n", err)
			continue
		}

		logClient.Infof("Create url file success: %s\n", url)
		logClient.Info(taskInfo)
	}

}

func handleCli(p *pikpak.PikPak) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			logClient.Errorf("Get parent id failed: %v\n", err)
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
		taskInfo, err := p.CreateUrlFile(parentId, url)
		if err != nil {
			logClient.Errorf("Create url file failed: %v\n", err)
			continue
		}
		logClient.Infof("Create url file success: ", url)

		logClient.Info(taskInfo)
	}
}
