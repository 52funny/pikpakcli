package link

import (
	"fmt"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/spf13/cobra"
)

var LinkCmd = &cobra.Command{
	Use:   "link",
	Short: `Print the direct download link of a file on the pikpak server`,
	Long: `Print the direct download link of a file stored on your pikpak drive.

The target is a file name (with -p as the base path) or a full remote path.
The link is the same signed URL that the pikpak web "download" action
generates, so it is time limited and must be fetched again later.`,
	Example: `  pikpakcli link -p Movies Peppa_Pig.mp4
  pikpakcli link /Movies/Peppa_Pig.mp4`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		if err := p.Login(); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		resolvedParentID := parentId
		if resolvedParentID == "" {
			id, err := p.GetPathFolderId(folder)
			if err != nil {
				return fmt.Errorf("get parent id failed: %w", err)
			}
			resolvedParentID = id
		}

		for _, arg := range args {
			stat, err := resolveLinkTarget(&p, resolvedParentID, arg)
			if err != nil {
				fmt.Println(arg, "get file stat error")
				logx.Error(err)
				continue
			}
			if stat.Kind == api.FileKindFolder {
				fmt.Println(stat.Name, "is a folder, folders have no download link")
				continue
			}

			file, err := p.GetFile(stat.ID)
			if err != nil {
				fmt.Println(arg, "get file error")
				logx.Error(err)
				continue
			}
			direct := directLink(&file)
			if direct == "" {
				fmt.Println(arg, "has no download link")
				continue
			}
			fmt.Println(direct)
		}
		return nil
	},
}

// Specifies the folder of the pikpak server
//
// default server root directory
var folder string

// parent path id
var parentId string

func init() {
	LinkCmd.Flags().StringVarP(&folder, "path", "p", "/", "specific the folder of the pikpak server")
	LinkCmd.Flags().StringVarP(&parentId, "parent-id", "P", "", "the parent path id")
}

func resolveLinkTarget(p *api.PikPak, resolvedParentID string, target string) (api.FileStat, error) {
	if strings.HasPrefix(target, "/") || strings.Contains(target, "/") {
		return p.GetFileByPath(target)
	}
	return p.GetFileStat(resolvedParentID, target)
}

// directLink picks the direct download URL of a file, preferring the original
// file link used by the pikpak web "download" action.
func directLink(file *api.File) string {
	if link := strings.TrimSpace(file.Links.ApplicationOctetStream.URL); link != "" {
		return link
	}
	return strings.TrimSpace(file.WebContentLink)
}
