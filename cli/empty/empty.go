package empty

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sync"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/spf13/cobra"
)

var targetPath string
var concurrency int
var deleteMode bool

type emptyFolderProvider interface {
	GetPathFolderId(dirPath string) (string, error)
	GetFolderFileStatList(parentId string) ([]api.FileStat, error)
	DeleteFile(fileId string) error
}

var EmptyCmd = &cobra.Command{
	Use:   "empty [path]",
	Short: "Recursively list empty folders on the PikPak server",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := targetPath
		if len(args) > 0 {
			path = args[0]
		}

		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		if err := p.Login(); err != nil {
			fmt.Println("Login failed")
			logx.Error(err)
			return
		}

		emptyFolders, err := handleEmptyFolders(cmd.Context(), &p, path, concurrency, deleteMode)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				fmt.Println("Empty folder scan canceled")
				return
			}
			fmt.Println("Handle empty folders failed")
			logx.Error(err)
			return
		}
		if len(emptyFolders) == 0 {
			fmt.Printf("No empty folders found under %s\n", path)
			return
		}
		for _, folder := range emptyFolders {
			if deleteMode {
				fmt.Printf("Deleted empty folder: %s\n", folder)
				continue
			}
			fmt.Printf("Empty folder: %s\n", folder)
		}
	},
}

func init() {
	EmptyCmd.Flags().StringVarP(&targetPath, "path", "p", "/", "The path where to remove empty folders recursively")
	EmptyCmd.Flags().IntVarP(&concurrency, "concurrency", "c", 8, "number of folders to process concurrently")
	EmptyCmd.Flags().BoolVarP(&deleteMode, "delete", "d", false, "delete the empty folders instead of only listing them")
}

func handleEmptyFolders(ctx context.Context, p emptyFolderProvider, rootPath string, concurrency int, deleteMode bool) ([]string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	rootID, err := p.GetPathFolderId(rootPath)
	if err != nil {
		return nil, err
	}
	if concurrency < 1 {
		concurrency = 1
	}

	deleted := make([]string, 0)
	state := emptyWalkState{
		sem: make(chan struct{}, concurrency),
	}
	if _, err := walkEmptyFolders(ctx, p, rootID, filepath.Clean(rootPath), filepath.Clean(rootPath) != string(filepath.Separator), deleteMode, &deleted, &state); err != nil {
		return nil, err
	}
	return deleted, nil
}

type emptyWalkState struct {
	sem chan struct{}
	mu  sync.Mutex
}

type emptyFolderResult struct {
	empty bool
	err   error
}

func walkEmptyFolders(ctx context.Context, p emptyFolderProvider, folderID, currentPath string, allowDeleteCurrent bool, deleteMode bool, deleted *[]string, state *emptyWalkState) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	files, err := p.GetFolderFileStatList(folderID)
	if err != nil {
		return false, err
	}

	hasFiles := false
	hasRemainingFolders := false
	results := make(chan emptyFolderResult, len(files))
	var childFolders int
	for _, file := range files {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if file.Kind != api.FileKindFolder {
			hasFiles = true
			continue
		}
		childFolders++

		childPath := filepath.Join(currentPath, file.Name)
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case state.sem <- struct{}{}:
			go func(file api.FileStat, childPath string) {
				defer func() {
					<-state.sem
				}()
				childEmpty, err := walkEmptyFolders(ctx, p, file.ID, childPath, true, deleteMode, deleted, state)
				results <- emptyFolderResult{
					empty: childEmpty,
					err:   err,
				}
			}(file, childPath)
		default:
			childEmpty, err := walkEmptyFolders(ctx, p, file.ID, childPath, true, deleteMode, deleted, state)
			results <- emptyFolderResult{
				empty: childEmpty,
				err:   err,
			}
		}
	}

	for i := 0; i < childFolders; i++ {
		select {
		case <-ctx.Done():
			return false, ctx.Err()
		case result := <-results:
			if result.err != nil {
				return false, result.err
			}
			if !result.empty {
				hasRemainingFolders = true
			}
		}
	}

	isEmpty := !hasFiles && !hasRemainingFolders
	if !isEmpty {
		return false, nil
	}
	if !allowDeleteCurrent {
		return true, nil
	}

	if deleteMode {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if err := p.DeleteFile(folderID); err != nil {
			return false, err
		}
	}
	state.mu.Lock()
	*deleted = append(*deleted, currentPath)
	state.mu.Unlock()
	return true, nil
}
