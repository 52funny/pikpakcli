package delete

import (
	"fmt"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var path string

var DeleteCmd = &cobra.Command{
	Use:     "delete [file-or-folder ...]",
	Aliases: []string{"del", "rm"},
	Short:   "Delete files or folders on the PikPak server",
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		p := api.NewPikPak(conf.Config.Username, conf.Config.Password)
		if err := p.Login(); err != nil {
			logrus.Errorf("Login failed: %v", err)
			return
		}

		flagPathSpecified := cmd.Flags().Changed("path")
		for parentPath, names := range groupDeleteTargets(args, flagPathSpecified) {
			if err := deleteEntries(&p, parentPath, names); err != nil {
				logrus.Error(err)
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

		if !forceParentPath {
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
			logrus.Errorf("Entry not found in %s: %s", parentPath, name)
			continue
		}

		if err := p.DeleteFile(file.ID); err != nil {
			logrus.Errorf("Delete %s from %s failed: %v", name, parentPath, err)
			continue
		}

		logrus.Infof("Deleted %s from %s", name, parentPath)
	}

	return nil
}
