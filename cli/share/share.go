package share

import (
	"fmt"
	"os"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/spf13/cobra"
)

var ShareCommand = &cobra.Command{
	Use:     "share",
	Aliases: []string{"d"},
	Short:   `Share file links on the pikpak server`,
	Run: func(cmd *cobra.Command, args []string) {
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			fmt.Println("Login failed")
			logx.Error(err)
			return
		}
		if len(args) > 0 {
			args, err = api.ExpandRemotePatterns(&p, folder, args, false)
			if err != nil {
				fmt.Println("Expand share target failed")
				logx.Error(err)
				return
			}
		}
		// Output file handle
		var f = os.Stdout
		if strings.TrimSpace(output) != "" {
			file, err := os.Create(output)
			if err != nil {
				fmt.Println("Create file failed")
				logx.Error(err)
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
func shareFolder(p *api.PikPak, f *os.File) {
	var err error
	if parentId == "" {
		parentId, err = p.GetDeepFolderId("", folder)
		if err != nil {
			fmt.Println("Get parent id failed")
			logx.Error(err)
			return
		}
	}
	fileStat, err := p.GetFolderFileStatList(parentId)
	if err != nil {
		fmt.Println("Get folder file stat list failed")
		logx.Error(err)
		return
	}
	for _, stat := range fileStat {
		// logrus.Debug(stat)
		if stat.Kind == api.FileKindFile {
			fmt.Fprintf(f, "PikPak://%s|%s|%s\n", stat.Name, stat.Size, stat.Hash)
		}
	}
}

// Share files
func shareFiles(p *api.PikPak, args []string, f *os.File) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(folder)
		if err != nil {
			fmt.Println("Get parent id failed")
			logx.Error(err)
			return
		}
	}
	for _, path := range args {
		stat, err := resolveShareTarget(p, parentId, path)
		if err != nil {
			fmt.Println(path, "get file stat error")
			logx.Error(err)
			continue
		}
		fmt.Fprintf(f, "PikPak://%s|%s|%s\n", stat.Name, stat.Size, stat.Hash)
	}
}

func resolveShareTarget(p *api.PikPak, resolvedParentID string, target string) (api.FileStat, error) {
	if strings.HasPrefix(target, "/") || strings.Contains(target, "/") {
		return p.GetFileByPath(target)
	}
	return p.GetFileStat(resolvedParentID, target)
}
