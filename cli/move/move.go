package move

import (
	"errors"
	"fmt"
	"path"
	"sort"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/spf13/cobra"
)

const batchSize = 100

var literalSources bool

type moveSourceProvider interface {
	GetPathFolderId(dirPath string) (string, error)
	GetFolderFileStatList(parentID string) ([]api.FileStat, error)
}

type mover interface {
	Move(fileIDs []string, parentID string) error
}

type moveBatchSummary struct {
	confirmedItems int
	failedItems    int
	failedBatches  int
}

type moveSelection struct {
	ids                  []string
	destinationSelf      []api.FileStat
	alreadyInDestination []api.FileStat
}

var MoveCmd = &cobra.Command{
	Use:     "mv <source>... <destination-folder>",
	Aliases: []string{"move"},
	Short:   "Move files or folders on the PikPak drive",
	Args:    cobra.MinimumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		if err := p.Login(); err != nil {
			return fmt.Errorf("login failed: %w", err)
		}

		destination := path.Clean(args[len(args)-1])
		destinationID, err := p.GetPathFolderId(destination)
		if err != nil {
			return fmt.Errorf("get destination folder %s: %w", destination, err)
		}

		sources := args[:len(args)-1]
		if !literalSources {
			sources, err = api.ExpandRemotePatterns(&p, "/", sources, false)
			if err != nil {
				return fmt.Errorf("expand move source: %w", err)
			}
		}

		files, err := resolveMoveSources(&p, sources)
		if err != nil {
			return err
		}
		selection := selectMoveSources(files, destinationID)
		for _, file := range selection.destinationSelf {
			fmt.Printf("Skipped destination folder itself: %s\n", file.Name)
		}
		for _, file := range selection.alreadyInDestination {
			fmt.Printf("Already in %s: %s\n", destination, file.Name)
		}

		summary, err := moveInBatches(&p, selection.ids, destinationID)
		if err != nil {
			logx.Error(err)
			return err
		}

		fmt.Printf("Moved %d item(s) to %s\n", summary.confirmedItems, destination)
		return nil
	},
}

func init() {
	MoveCmd.Flags().BoolVar(&literalSources, "literal", false, "treat source paths as literal names instead of wildcard patterns")
}

func resolveMoveSources(p moveSourceProvider, sources []string) ([]api.FileStat, error) {
	byParent := make(map[string][]string)
	for _, source := range sources {
		cleanSource := path.Clean(source)
		parent := path.Dir(cleanSource)
		if parent == "." {
			parent = "/"
		}
		byParent[parent] = append(byParent[parent], path.Base(cleanSource))
	}

	parents := make([]string, 0, len(byParent))
	for parent := range byParent {
		parents = append(parents, parent)
	}
	sort.Strings(parents)

	resolved := make([]api.FileStat, 0, len(sources))
	for _, parent := range parents {
		parentID, err := p.GetPathFolderId(parent)
		if err != nil {
			return nil, fmt.Errorf("get source folder %s: %w", parent, err)
		}
		entries, err := p.GetFolderFileStatList(parentID)
		if err != nil {
			return nil, fmt.Errorf("list source folder %s: %w", parent, err)
		}
		index := make(map[string]api.FileStat, len(entries))
		for _, entry := range entries {
			index[entry.Name] = entry
		}
		for _, name := range byParent[parent] {
			entry, ok := index[name]
			if !ok {
				return nil, fmt.Errorf("move source not found: %s", path.Join(parent, name))
			}
			resolved = append(resolved, entry)
		}
	}
	return resolved, nil
}

func selectMoveSources(files []api.FileStat, destinationID string) moveSelection {
	selection := moveSelection{ids: make([]string, 0, len(files))}
	seen := make(map[string]struct{}, len(files))
	for _, file := range files {
		if file.ID == destinationID {
			selection.destinationSelf = append(selection.destinationSelf, file)
			continue
		}
		if file.ParentID == destinationID {
			selection.alreadyInDestination = append(selection.alreadyInDestination, file)
			continue
		}
		if _, ok := seen[file.ID]; ok {
			continue
		}
		seen[file.ID] = struct{}{}
		selection.ids = append(selection.ids, file.ID)
	}
	return selection
}

func moveInBatches(p mover, ids []string, destinationID string) (moveBatchSummary, error) {
	var summary moveBatchSummary
	var failures []error

	for start := 0; start < len(ids); start += batchSize {
		end := start + batchSize
		if end > len(ids) {
			end = len(ids)
		}

		batchNumber := start/batchSize + 1
		batch := ids[start:end]
		if err := p.Move(batch, destinationID); err != nil {
			summary.failedItems += len(batch)
			summary.failedBatches++
			failures = append(failures, fmt.Errorf("batch %d (%d item(s)): %w", batchNumber, len(batch), err))
			continue
		}
		summary.confirmedItems += len(batch)
	}

	if len(failures) > 0 {
		return summary, fmt.Errorf(
			"move completed with failures: %d item(s) confirmed moved; %d item(s) in %d failed batch(es): %w",
			summary.confirmedItems,
			summary.failedItems,
			summary.failedBatches,
			errors.Join(failures...),
		)
	}

	return summary, nil
}
