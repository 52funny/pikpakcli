package sha

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

var NewShaCommand = &cobra.Command{
	Use:   "sha",
	Short: `Create a file according to sha`,
	Run: func(cmd *cobra.Command, args []string) {
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			fmt.Println("Login failed")
			logx.Error(err)
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
			handleNewSha(&p, shas)
			return
		}

		// args mode
		if len(args) > 0 {
			handleNewSha(&p, args)
		} else {
			fmt.Println("Please input the folder name")
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
func handleNewSha(p *api.PikPak, shas []string) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(path)
		if err != nil {
			fmt.Println("Get parent id failed")
			logx.Error(err)
			return
		}
	}

	for _, sha := range shas {
		sha = sha[strings.Index(sha, "://")+3:]
		shaElements := strings.Split(sha, "|")
		if len(shaElements) != 3 {
			fmt.Println("The sha format is wrong:", sha)
			continue
		}
		name, size, sha := shaElements[0], shaElements[1], shaElements[2]
		err := p.CreateShaFile(parentId, name, size, sha)
		if err != nil {
			fmt.Println("Create sha file failed")
			logx.Error(err)
			continue
		}
		fmt.Println("Create sha file success:", name)
	}
}
