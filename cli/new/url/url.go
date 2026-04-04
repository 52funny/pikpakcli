package url

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/spf13/cobra"
)

var NewUrlCommand = &cobra.Command{
	Use:   "url",
	Short: `Create a file according to url`,
	Run: func(cmd *cobra.Command, args []string) {
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			fmt.Println("Login failed")
			logx.Error(err)
			return
		}
		if cli {
			handleCli(&p)
			return
		}
		// input mode
		if strings.TrimSpace(input) != "" {
			f, err := os.OpenFile(input, os.O_RDONLY, 0666)
			if err != nil {
				fmt.Printf("Open file %s failed\n", input)
				logx.Error(err)
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
			fmt.Println("Please input the folder name")
		}
	},
}

var path string

var parentId string

var input string

var cli bool

func init() {
	NewUrlCommand.Flags().StringVarP(&path, "path", "p", "/", "The path of the folder")
	NewUrlCommand.Flags().StringVarP(&parentId, "parent-id", "P", "", "The parent id")
	NewUrlCommand.Flags().StringVarP(&input, "input", "i", "", "The input of the sha file")
	NewUrlCommand.Flags().BoolVarP(&cli, "cli", "c", false, "The cli mode")
}

// new folder
func handleNewUrl(p *api.PikPak, shas []string) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			fmt.Println("Get parent id failed")
			logx.Error(err)
			return
		}
	}
	for _, url := range shas {
		err := p.CreateUrlFile(parentId, url)
		if err != nil {
			fmt.Println("Create url file failed")
			logx.Error(err)
			continue
		}
		fmt.Println("Create url file success:", url)
	}
}

func handleCli(p *api.PikPak) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			fmt.Println("Get parent id failed")
			logx.Error(err)
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
		err = p.CreateUrlFile(parentId, url)
		if err != nil {
			fmt.Println("Create url file failed")
			logx.Error(err)
			continue
		}
		fmt.Println("Create url file success:", url)
	}
}
