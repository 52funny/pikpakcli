package move

import (
	"errors"
	"fmt"
	"testing"

	"github.com/52funny/pikpakcli/internal/api"
	"github.com/stretchr/testify/require"
)

type fakeMoveSourceProvider struct {
	ids       map[string]string
	entries   map[string][]api.FileStat
	requested []string
}

func TestMoveCommandUsesMVAsPrimaryName(t *testing.T) {
	require.Equal(t, "mv", MoveCmd.Name())
	require.Contains(t, MoveCmd.Aliases, "move")
}

func (f *fakeMoveSourceProvider) GetPathFolderId(dirPath string) (string, error) {
	f.requested = append(f.requested, dirPath)
	id, ok := f.ids[dirPath]
	if !ok {
		return "", fmt.Errorf("folder not found: %s", dirPath)
	}
	return id, nil
}

func (f *fakeMoveSourceProvider) GetFolderFileStatList(parentID string) ([]api.FileStat, error) {
	return f.entries[parentID], nil
}

func TestResolveMoveSourcesTreatsBareNameAsRootRelative(t *testing.T) {
	provider := &fakeMoveSourceProvider{
		ids: map[string]string{"/": "root-id"},
		entries: map[string][]api.FileStat{
			"root-id": {{ID: "file-id", Name: "movie.mkv"}},
		},
	}

	files, err := resolveMoveSources(provider, []string{"movie.mkv"})

	require.NoError(t, err)
	require.Equal(t, []string{"/"}, provider.requested)
	require.Equal(t, []api.FileStat{{ID: "file-id", Name: "movie.mkv"}}, files)
}

func TestResolveMoveSourcesHandlesNestedAndAbsoluteParents(t *testing.T) {
	provider := &fakeMoveSourceProvider{
		ids: map[string]string{
			"/Movies": "movies-id",
			"TV":      "tv-id",
		},
		entries: map[string][]api.FileStat{
			"movies-id": {{ID: "movie-id", Name: "movie.mkv"}},
			"tv-id":     {{ID: "show-id", Name: "show.mkv"}},
		},
	}

	files, err := resolveMoveSources(provider, []string{"TV/show.mkv", "/Movies/movie.mkv"})

	require.NoError(t, err)
	require.Equal(t, []string{"/Movies", "TV"}, provider.requested)
	require.Equal(t, []string{"movie-id", "show-id"}, []string{files[0].ID, files[1].ID})
}

func TestResolveMoveSourcesReportsMissingSource(t *testing.T) {
	provider := &fakeMoveSourceProvider{
		ids:     map[string]string{"/": "root-id"},
		entries: map[string][]api.FileStat{"root-id": nil},
	}

	_, err := resolveMoveSources(provider, []string{"missing.txt"})

	require.EqualError(t, err, "move source not found: /missing.txt")
}

func TestSelectMoveSourcesSkipsDestinationAlreadyMovedAndDuplicates(t *testing.T) {
	selection := selectMoveSources([]api.FileStat{
		{ID: "destination", Name: "Archive"},
		{ID: "already-there", ParentID: "destination", Name: "existing.txt"},
		{ID: "move-me", ParentID: "source", Name: "movie.mkv"},
		{ID: "move-me", ParentID: "source", Name: "movie.mkv"},
	}, "destination")

	require.Equal(t, []string{"move-me"}, selection.ids)
	require.Equal(t, []string{"destination"}, []string{selection.destinationSelf[0].ID})
	require.Equal(t, []string{"already-there"}, []string{selection.alreadyInDestination[0].ID})
}

type recordingMover struct {
	calls  [][]string
	failAt map[int]error
}

func (m *recordingMover) Move(ids []string, _ string) error {
	call := len(m.calls)
	m.calls = append(m.calls, append([]string(nil), ids...))
	return m.failAt[call]
}

func TestMoveInBatchesContinuesAndAggregatesFailures(t *testing.T) {
	ids := make([]string, 205)
	for i := range ids {
		ids[i] = fmt.Sprintf("id-%03d", i)
	}
	mover := &recordingMover{failAt: map[int]error{
		0: errors.New("first failed"),
		2: errors.New("third failed"),
	}}

	summary, err := moveInBatches(mover, ids, "destination")

	require.Error(t, err)
	require.Len(t, mover.calls, 3)
	require.Len(t, mover.calls[0], 100)
	require.Len(t, mover.calls[1], 100)
	require.Len(t, mover.calls[2], 5)
	require.Equal(t, moveBatchSummary{confirmedItems: 100, failedItems: 105, failedBatches: 2}, summary)
	require.ErrorContains(t, err, "100 item(s) confirmed moved")
	require.ErrorContains(t, err, "105 item(s) in 2 failed batch(es)")
	require.ErrorContains(t, err, "batch 1 (100 item(s)): first failed")
	require.ErrorContains(t, err, "batch 3 (5 item(s)): third failed")
}

func TestMoveInBatchesReportsAllConfirmedItems(t *testing.T) {
	mover := &recordingMover{}

	summary, err := moveInBatches(mover, []string{"one", "two"}, "destination")

	require.NoError(t, err)
	require.Equal(t, moveBatchSummary{confirmedItems: 2}, summary)
	require.Equal(t, [][]string{{"one", "two"}}, mover.calls)
}
