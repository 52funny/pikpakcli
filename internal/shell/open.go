package shell

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/utils"
)

const (
	openCategoryDefault = "default"
	openCategoryText    = "text"
	openCategoryImage   = "image"
	openCategoryVideo   = "video"
	openCategoryAudio   = "audio"
	openCategoryPDF     = "pdf"
)

type openFileService interface {
	GetFileByPath(path string) (api.FileStat, error)
	GetFile(fileID string) (api.File, error)
}

func handleOpenCommand(p openFileService, currentPath string, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: open <remote-file> [remote-file...]")
	}

	for _, arg := range args {
		targetPath := resolveShellPath(currentPath, arg)
		stat, err := p.GetFileByPath(targetPath)
		if err != nil {
			return fmt.Errorf("open: %s: %w", targetPath, err)
		}
		if stat.Kind == api.FileKindFolder {
			return fmt.Errorf("open: %s: folders are not supported", targetPath)
		}

		file, err := p.GetFile(stat.ID)
		if err != nil {
			return fmt.Errorf("open: %s: get file failed: %w", targetPath, err)
		}

		openTarget, err := resolveOpenTarget(&file)
		if err != nil {
			return fmt.Errorf("open: %s: resolve open target failed: %w", targetPath, err)
		}

		if err := openWithLocalApp(openTarget, classifyOpenCategory(file.Name)); err != nil {
			return fmt.Errorf("open: %s: launch local app failed: %w", targetPath, err)
		}

		fmt.Printf("Opened %s -> %s\n", targetPath, openTarget)
	}

	return nil
}

func resolveOpenTarget(file *api.File) (string, error) {
	if classifyOpenCategory(file.Name) == openCategoryVideo {
		if url := remoteVideoOpenURL(file); url != "" {
			return url, nil
		}
	}

	return cacheOpenFile(file)
}

func cacheOpenFile(file *api.File) (string, error) {
	cacheRoot, err := openCacheRoot()
	if err != nil {
		return "", err
	}

	cacheDir := filepath.Join(cacheRoot, file.ID)
	if err := utils.CreateDirIfNotExist(cacheDir); err != nil {
		return "", err
	}

	localPath := filepath.Join(cacheDir, file.Name)
	matched, err := localFileMatchesRemoteSize(localPath, file.Size)
	if err != nil {
		return "", err
	}
	if matched {
		return localPath, nil
	}

	if err := file.Download(localPath, nil); err != nil {
		return "", err
	}

	return localPath, nil
}

func openCacheRoot() (string, error) {
	if strings.TrimSpace(conf.Config.Open.DownloadDir) != "" {
		root := utils.ExpandLocalPath(conf.Config.Open.DownloadDir)
		if err := utils.CreateDirIfNotExist(root); err != nil {
			return "", err
		}
		return root, nil
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = filepath.Join(os.TempDir(), "pikpakcli")
	}

	root := filepath.Join(cacheDir, "pikpakcli", "open")
	if err := utils.CreateDirIfNotExist(root); err != nil {
		return "", err
	}
	return root, nil
}

func localFileMatchesRemoteSize(path string, remoteSize string) (bool, error) {
	expectedSize, err := strconv.ParseInt(remoteSize, 10, 64)
	if err != nil || expectedSize < 0 {
		return false, nil
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return info.Size() == expectedSize, nil
}

func classifyOpenCategory(name string) string {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".txt", ".md", ".markdown", ".log", ".json", ".yaml", ".yml", ".toml", ".ini", ".cfg", ".conf", ".csv",
		".go", ".rs", ".py", ".js", ".ts", ".tsx", ".jsx", ".java", ".c", ".cc", ".cpp", ".h", ".hpp", ".sh", ".zsh":
		return openCategoryText
	case ".jpg", ".jpeg", ".png", ".gif", ".bmp", ".webp", ".svg", ".heic", ".tiff":
		return openCategoryImage
	case ".mp4", ".mkv", ".mov", ".avi", ".wmv", ".flv", ".webm", ".m4v":
		return openCategoryVideo
	case ".mp3", ".flac", ".wav", ".aac", ".m4a", ".ogg", ".opus":
		return openCategoryAudio
	case ".pdf":
		return openCategoryPDF
	default:
		return openCategoryDefault
	}
}

func remoteVideoOpenURL(file *api.File) string {
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
	if strings.TrimSpace(file.Links.ApplicationOctetStream.URL) != "" {
		return file.Links.ApplicationOctetStream.URL
	}
	if strings.TrimSpace(file.WebContentLink) != "" {
		return file.WebContentLink
	}
	return ""
}

func openWithLocalApp(target string, category string) error {
	name, args, err := buildOpenCommand(runtime.GOOS, conf.Config.Open, target, category)
	if err != nil {
		return err
	}

	cmd := exec.Command(name, args...)
	return cmd.Start()
}

func buildOpenCommand(goos string, cfg conf.OpenConfig, path string, category string) (string, []string, error) {
	command := commandForCategory(cfg, category)
	if len(command) == 0 {
		command = defaultOpenCommand(goos, category)
	}
	if len(command) == 0 {
		return "", nil, fmt.Errorf("unsupported platform: %s", goos)
	}

	resolved := make([]string, 0, len(command)+1)
	hasPlaceholder := false
	for _, item := range command {
		if item == "{path}" {
			resolved = append(resolved, path)
			hasPlaceholder = true
			continue
		}
		resolved = append(resolved, item)
	}
	if !hasPlaceholder {
		resolved = append(resolved, path)
	}

	return resolved[0], resolved[1:], nil
}

func commandForCategory(cfg conf.OpenConfig, category string) []string {
	switch category {
	case openCategoryText:
		if len(cfg.Text) > 0 {
			return append([]string{}, cfg.Text...)
		}
	case openCategoryImage:
		if len(cfg.Image) > 0 {
			return append([]string{}, cfg.Image...)
		}
	case openCategoryVideo:
		if len(cfg.Video) > 0 {
			return append([]string{}, cfg.Video...)
		}
	case openCategoryAudio:
		if len(cfg.Audio) > 0 {
			return append([]string{}, cfg.Audio...)
		}
	case openCategoryPDF:
		if len(cfg.PDF) > 0 {
			return append([]string{}, cfg.PDF...)
		}
	}

	if len(cfg.Default) > 0 {
		return append([]string{}, cfg.Default...)
	}
	return nil
}

func defaultOpenCommand(goos string, category string) []string {
	switch goos {
	case "darwin":
		switch category {
		case openCategoryText:
			return []string{"open", "-a", "TextEdit"}
		case openCategoryImage, openCategoryPDF:
			return []string{"open", "-a", "Preview"}
		case openCategoryVideo, openCategoryAudio:
			return []string{"open", "-a", "IINA"}
		default:
			return []string{"open"}
		}
	case "linux":
		return []string{"xdg-open"}
	case "windows":
		return []string{"cmd", "/c", "start", ""}
	default:
		return nil
	}
}
