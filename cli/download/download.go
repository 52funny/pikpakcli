package download

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
)

var DownloadCmd = &cobra.Command{
	Use:     "download",
	Aliases: []string{"d"},
	Short:   `Download file from pikpak server`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			fmt.Println("Login failed")
			logx.Error(err)
			return
		}
		args, err = api.ExpandRemotePatterns(&p, folder, args, false)
		if err != nil {
			fmt.Println("Expand download target failed")
			logx.Error(err)
			return
		}
		handleDownload(cmd, &p, args)
	},
}

// Number of simultaneous downloads
//
// default 1
var count int

// Specifies the folder of the pikpak server
//
// default server root directory (.)
var folder string

// parent path id
var parentId string

// Output directory
//
// default current directory (.)
var output string

// Progress bar
//
// default false
var progress bool

// Time range
//
// default empty, download the whole file
var timeRangeSpec string

var clipper timeClipper = ffmpegClipper{}

type warpFile struct {
	f      *api.File
	output string
}

type warpStat struct {
	s      api.FileStat
	output string
}

const progressNameMaxRunes = 36

func init() {
	DownloadCmd.Flags().IntVarP(&count, "count", "c", 1, "number of simultaneous downloads")
	DownloadCmd.Flags().StringVarP(&output, "output", "o", ".", "output directory")
	DownloadCmd.Flags().StringVarP(&folder, "path", "p", "/", "specific the base path on the pikpak server")
	DownloadCmd.Flags().StringVarP(&parentId, "parent-id", "P", "", "the parent path id")
	DownloadCmd.Flags().BoolVarP(&progress, "progress", "g", false, "show download progress")
	DownloadCmd.Flags().StringVar(&timeRangeSpec, "time-range", "", "download a time range from a single video file, for example 10-20 or 01:02-01:30")
}

type downloadTargetResolver interface {
	GetFileByPath(path string) (api.FileStat, error)
	GetFileStat(parentId string, name string) (api.FileStat, error)
	GetPathFolderId(dirPath string) (string, error)
}

func handleDownload(cmd *cobra.Command, p *api.PikPak, args []string) {
	if err := utils.CreateDirIfNotExist(output); err != nil {
		fmt.Println("Create output directory failed")
		logx.Error(err)
		return
	}

	if strings.TrimSpace(timeRangeSpec) != "" {
		downloadTimeRangeTarget(p, args)
		return
	}

	if requiresExplicitOutputFlag(cmd, args) {
		fmt.Println("Use -o to specify the output directory when downloading specific files")
		return
	}

	for _, arg := range args {
		downloadTarget(p, arg)
	}
}

func requiresExplicitOutputFlag(cmd *cobra.Command, args []string) bool {
	if cmd.Flags().Changed("output") || len(args) <= 1 {
		return false
	}
	for _, arg := range args {
		trimmed := strings.TrimSpace(arg)
		if trimmed == "." || trimmed == ".." {
			return true
		}
	}
	return false
}

func downloadTimeRangeTarget(p *api.PikPak, args []string) {
	if len(args) != 1 {
		fmt.Println("Time range download requires exactly one file target")
		return
	}

	timeRange, err := parseTimeRangeSpec(timeRangeSpec)
	if err != nil {
		fmt.Println("Parse time range failed:", err)
		return
	}

	stat, err := resolveDownloadTarget(p, args[0])
	if err != nil {
		target := remoteTargetPath(args[0])
		fmt.Println("Resolve download target failed:", target)
		logx.Error(err)
		return
	}
	if stat.Kind == api.FileKindFolder {
		fmt.Println("Time range download only supports file targets")
		return
	}

	file := mustGetFile(p, stat)
	sourceURL := mediaClipSourceURL(file)
	if sourceURL == "" {
		fmt.Println("Time range download failed: file has no playable media URL")
		return
	}

	path := filepath.Join(output, timeRangeOutputName(file.Name, timeRange))
	if err := clipper.Clip(sourceURL, timeRange, path); err != nil {
		fmt.Println("Time range download failed:", file.Name)
		logx.Error(err)
		return
	}
	fmt.Println("Download", file.Name, "Success")
}

type TimeRange struct {
	Start string
	End   string
}

type timeClipper interface {
	Clip(sourceURL string, timeRange TimeRange, outputPath string) error
}

type ffmpegClipper struct{}

func (ffmpegClipper) Clip(sourceURL string, timeRange TimeRange, outputPath string) error {
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg is required for --time-range; install ffmpeg and make sure it is in PATH")
	}

	args := []string{
		"-y",
		"-ss", timeRange.Start,
	}
	if timeRange.End != "" {
		args = append(args, "-to", timeRange.End)
	}
	args = append(args, "-i", sourceURL, "-c", "copy", outputPath)

	cmd := exec.Command(ffmpegPath, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func parseTimeRangeSpec(spec string) (TimeRange, error) {
	trimmed := strings.TrimSpace(spec)
	if trimmed == "" {
		return TimeRange{}, fmt.Errorf("time range must not be empty")
	}

	parts := strings.Split(trimmed, "-")
	if len(parts) != 2 {
		return TimeRange{}, fmt.Errorf("time range must be START-END or START-")
	}
	if strings.TrimSpace(parts[0]) == "" {
		return TimeRange{}, fmt.Errorf("time range start must not be empty")
	}

	start := strings.TrimSpace(parts[0])
	end := strings.TrimSpace(parts[1])
	if err := validateTimePoint(start); err != nil {
		return TimeRange{}, fmt.Errorf("invalid time range start: %w", err)
	}
	if end != "" {
		if err := validateTimePoint(end); err != nil {
			return TimeRange{}, fmt.Errorf("invalid time range end: %w", err)
		}
	}
	return TimeRange{Start: start, End: end}, nil
}

func validateTimePoint(value string) error {
	if value == "" {
		return fmt.Errorf("time point must not be empty")
	}
	for _, part := range strings.Split(value, ":") {
		if part == "" {
			return fmt.Errorf("time point contains an empty component")
		}
		n, err := strconv.ParseInt(part, 10, 64)
		if err != nil || n < 0 {
			return fmt.Errorf("time point components must be non-negative integers")
		}
	}
	return nil
}

func mediaClipSourceURL(file *api.File) string {
	for _, media := range file.Medias {
		if media.IsDefault && media.IsVisible && strings.TrimSpace(media.Link.URL) != "" {
			return media.Link.URL
		}
	}
	for _, media := range file.Medias {
		if media.IsVisible && strings.TrimSpace(media.Link.URL) != "" {
			return media.Link.URL
		}
	}
	for _, media := range file.Medias {
		if strings.TrimSpace(media.Link.URL) != "" {
			return media.Link.URL
		}
	}
	return strings.TrimSpace(file.Links.ApplicationOctetStream.URL)
}

func timeRangeOutputName(name string, timeRange TimeRange) string {
	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	suffix := strings.ReplaceAll(timeRange.Start, ":", "-") + "-"
	if timeRange.End != "" {
		suffix += strings.ReplaceAll(timeRange.End, ":", "-")
	} else {
		suffix += "end"
	}
	return base + "." + suffix + ext
}

func downloadTarget(p *api.PikPak, arg string) {
	stat, err := resolveDownloadTarget(p, arg)
	if err != nil {
		target := remoteTargetPath(arg)
		fmt.Println("Resolve download target failed:", target)
		logx.Error(err)
		return
	}

	if stat.Kind == api.FileKindFolder {
		downloadFolder(p, stat.ID, localOutputRoot(stat.Name))
		return
	}

	downloadFiles(p, []warpFile{
		{
			f:      mustGetFile(p, stat),
			output: output,
		},
	})
}

func downloadFolder(p *api.PikPak, folderID string, rootOutput string) {
	collectStat := make([]warpStat, 0)
	recursive(p, &collectStat, folderID, rootOutput)
	downloadStats(p, collectStat)
}

func downloadStats(p *api.PikPak, collectStat []warpStat) {
	if len(collectStat) == 0 {
		return
	}

	statCh := make(chan warpStat, len(collectStat))
	statDone := make(chan struct{})

	fileCh := make(chan warpFile, len(collectStat))
	fileDone := make(chan struct{})

	for i := 0; i < 4; i += 1 {
		go func(fileCh chan<- warpFile, statCh <-chan warpStat, statDone chan<- struct{}) {
			for {
				stat, ok := <-statCh
				if !ok {
					break
				}
				file, err := p.GetFile(stat.s.ID)
				if err != nil {
					fmt.Println("Get file failed")
					logx.Error(err)
				}
				fileCh <- warpFile{
					f:      &file,
					output: stat.output,
				}
				statDone <- struct{}{}
			}
		}(fileCh, statCh, statDone)
	}

	pb := startDownloadWorkers(fileCh, fileDone)

	for i := 0; i < len(collectStat); i += 1 {
		err := utils.CreateDirIfNotExist(collectStat[i].output)
		if err != nil {
			fmt.Println("Create output directory failed")
			logx.Error(err)
			return
		}
		statCh <- collectStat[i]
	}
	close(statCh)

	for i := 0; i < len(collectStat); i += 1 {
		<-statDone
	}
	close(statDone)

	for i := 0; i < len(collectStat); i += 1 {
		<-fileDone
	}
	if pb != nil {
		pb.Wait()
	}
}

func recursive(p *api.PikPak, collectWarpFile *[]warpStat, parentId string, parentPath string) {
	statList, err := p.GetFolderFileStatList(parentId)
	if err != nil {
		fmt.Println("Get folder file stat list failed")
		logx.Error(err)
		return
	}
	for _, r := range statList {
		if r.Kind == api.FileKindFolder {
			recursive(p, collectWarpFile, r.ID, filepath.Join(parentPath, r.Name))
		} else {
			// file, _ := p.GetFile(r.ID)
			*collectWarpFile = append(*collectWarpFile, warpStat{
				s:      r,
				output: parentPath,
			})
			// fmt.Println(r.Name, r.Size, r.Kind, parentPath)
		}
	}
}

func downloadFiles(p *api.PikPak, files []warpFile) {
	sendCh := make(chan warpFile, len(files))
	receiveCh := make(chan struct{}, len(files))
	pb := startDownloadWorkers(sendCh, receiveCh)
	for _, file := range files {
		sendCh <- file
	}
	close(sendCh)
	for i := 0; i < len(files); i++ {
		<-receiveCh
	}
	close(receiveCh)
	if pb != nil {
		pb.Wait()
	}
}

func startDownloadWorkers(sendCh <-chan warpFile, receiveCh chan<- struct{}) *mpb.Progress {
	var pb *mpb.Progress
	if progress {
		pb = mpb.New(
			mpb.WithWidth(30),
			mpb.WithAutoRefresh(),
		)
	}

	for i := 0; i < count; i++ {
		go download(sendCh, receiveCh, pb)
	}

	return pb
}

func resolveDownloadTarget(p downloadTargetResolver, arg string) (api.FileStat, error) {
	if target := strings.TrimSpace(arg); target == "" {
		if parentId != "" {
			return api.FileStat{
				Kind: api.FileKindFolder,
				ID:   parentId,
				Name: filepath.Base(filepath.Clean(folder)),
			}, nil
		}
		remotePath := remoteTargetPath("")
		if remotePath == string(filepath.Separator) {
			id, err := p.GetPathFolderId(folder)
			if err != nil {
				return api.FileStat{}, err
			}
			return api.FileStat{
				Kind: api.FileKindFolder,
				ID:   id,
				Name: "",
			}, nil
		}
		return p.GetFileByPath(remotePath)
	}

	if parentId != "" && !filepath.IsAbs(arg) && !strings.Contains(arg, string(filepath.Separator)) {
		return p.GetFileStat(parentId, arg)
	}

	return p.GetFileByPath(remoteTargetPath(arg))
}

func remoteTargetPath(arg string) string {
	base := strings.TrimSpace(folder)
	target := strings.TrimSpace(arg)
	if target == "" {
		target = "."
	}
	if filepath.IsAbs(target) {
		return filepath.Clean(target)
	}
	return filepath.Clean(filepath.Join(string(filepath.Separator), base, target))
}

func localOutputRoot(name string) string {
	if strings.TrimSpace(name) == "" || name == string(filepath.Separator) || name == "." {
		return output
	}
	return filepath.Join(output, name)
}

func mustGetFile(p *api.PikPak, stat api.FileStat) *api.File {
	file, err := p.GetFile(stat.ID)
	if err != nil {
		fmt.Println("Get file failed")
		logx.Error(err)
		return &api.File{FileStat: stat}
	}
	return &file
}

func progressDisplayName(warp warpFile) string {
	name := warp.f.Name
	if base := filepath.Base(filepath.Clean(warp.output)); base != "." && base != string(filepath.Separator) && base != "" {
		name = filepath.Join(base, name)
	}
	return trimRunes(name, progressNameMaxRunes)
}

func trimRunes(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

func download(inCh <-chan warpFile, out chan<- struct{}, pb *mpb.Progress) {
	for {
		warp, ok := <-inCh
		if !ok {
			break
		}

		path := filepath.Join(warp.output, warp.f.Name)
		exist, err := utils.Exists(path)
		if err != nil {
			// logrus.Errorln("Access", path, "Failed:", err)
			out <- struct{}{}
			continue
		}
		flag := path + ".pikpakclidownload"
		hasFlag, err := utils.Exists(flag)
		if err != nil {
			// logrus.Errorln("Access", flag, "Failed:", err)
			out <- struct{}{}
			continue
		}
		if exist && !hasFlag {
			// logrus.Infoln("Skip downloaded file", warp.f.Name)
			out <- struct{}{}
			continue
		}
		err = utils.TouchFile(flag)
		if err != nil {
			// logrus.Errorln("Create flag file", flag, "Failed:", err)
			out <- struct{}{}
			continue
		}

		siz, err := strconv.ParseInt(warp.f.Size, 10, 64)
		if err != nil {
			// logrus.Errorln("Parse File size", warp.f.Size, "Failed:", err)
			out <- struct{}{}
			continue
		}

		var bar *mpb.Bar

		if pb != nil {
			bar = pb.AddBar(siz,
				mpb.PrependDecorators(
					decor.Name(progressDisplayName(warp), decor.WC{W: progressNameMaxRunes + 2, C: decor.DSyncWidth}),
					decor.CountersKibiByte("% .1f / % .1f", decor.WCSyncSpace),
					decor.Percentage(decor.WCSyncSpace),
				),
				mpb.AppendDecorators(
					decor.Name(" | ", decor.WCSyncSpace),
					decor.Name("ETA ", decor.WCSyncSpace),
					decor.EwmaETA(decor.ET_STYLE_GO, 30),
					decor.Name(" | ", decor.WCSyncSpace),
					decor.Name("SPD ", decor.WCSyncSpace),
					decor.EwmaSpeed(decor.SizeB1024(0), "% .2f", 60),
				),
			)
		}

		// start downloading
		err = warp.f.Download(path, bar)
		// if hasn't error then remove flag file
		if err == nil {
			if pb == nil {
				fmt.Println("Download", warp.f.Name, "Success")
			}
			os.Remove(flag)
			if bar != nil {
				bar.SetTotal(siz, true)
			}
		} else {
			if pb == nil {
				fmt.Println("Download failed:", warp.f.Name)
				logx.Error(err)
			}
			if bar != nil {
				bar.Abort(false)
			}
		}
		out <- struct{}{}
	}
}
