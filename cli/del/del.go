package delete

import (
	"fmt"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/spf13/cobra"
)

var path string

var DeleteCmd = &cobra.Command{
	Use:     "delete [file-or-folder ...]",
	Aliases: []string{"del", "rm"},
	Short:   "Delete files or folders on the PikPak server",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		if err := p.Login(); err != nil {
			fmt.Println("Login failed")
			logx.Error(err)
			return
		}

		flagPathSpecified := cmd.Flags().Changed("path")
		args, err := api.ExpandRemotePatterns(&p, path, args, flagPathSpecified)
		if err != nil {
			fmt.Println("Expand delete target failed")
			logx.Error(err)
			return
		}
		for parentPath, names := range groupDeleteTargets(args, flagPathSpecified) {
			if err := deleteEntries(&p, parentPath, names); err != nil {
				fmt.Println("Delete entries failed")
				logx.Error(err)
			}
		}
	},
}

func init() {
	DeleteCmd.Flags().StringVarP(&path, "path", "p", "/", "The path where to look for the file")
}

func groupDeleteTargets(args []string, forceParentPath bool) map[string][]string {
	targets := make(map[string][]string)
	for _, arg := range args {
		parentPath := path
		name := arg

		if !forceParentPath || strings.HasPrefix(arg, "/") || strings.Contains(arg, "/") {
			resolvedParentPath, resolvedName := utils.SplitRemotePath(arg)
			if resolvedName == "" {
				continue
			}
			name = resolvedName
			if resolvedParentPath == "" {
				parentPath = "/"
			} else {
				parentPath = resolvedParentPath
			}
		}

		targets[parentPath] = append(targets[parentPath], name)
	}
	return targets
}

func deleteEntries(p *api.PikPak, parentPath string, names []string) error {
	parentID, err := p.GetPathFolderId(parentPath)
	if err != nil {
		return fmt.Errorf("get path folder id for %s failed: %w", parentPath, err)
	}

	files, err := p.GetFolderFileStatList(parentID)
	if err != nil {
		return fmt.Errorf("get file list for %s failed: %w", parentPath, err)
	}

	fileIndex := make(map[string]api.FileStat, len(files))
	for _, file := range files {
		fileIndex[file.Name] = file
	}

	for _, name := range names {
		file, ok := fileIndex[name]
		if !ok {
			fmt.Printf("Entry not found in %s: %s\n", parentPath, name)
			continue
		}

		if err := p.DeleteFile(file.ID); err != nil {
			fmt.Printf("Delete %s from %s failed\n", name, parentPath)
			logx.Error(err)
			continue
		}

		fmt.Printf("Deleted %s from %s\n", name, parentPath)
	}

	return nil
}
