package upload

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/api"
	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/spf13/cobra"
)

var UploadCmd = &cobra.Command{
	Use:     "upload",
	Aliases: []string{"u"},
	Short:   `Upload file to pikpak server`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) == 0 {
			cmd.Help()
			return
		}
		api.Concurrent = uploadConcurrency
		p := api.NewPikPakWithContext(cmd.Context(), conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			fmt.Println("Login failed")
			logx.Error(err)
			return
		}
		err = p.AuthCaptchaToken("POST:/drive/v1/files")
		if err != nil {
			fmt.Println("Auth captcha token failed")
			logx.Error(err)
			return
		}

		go func() {
			ticker := time.NewTicker(time.Second * 7200 * 3 / 4)
			defer ticker.Stop()
			for range ticker.C {
				err := p.RefreshToken()
				if err != nil {
					logx.Warn("session", "refresh token failed:", err)
					continue
				}
			}
		}()
		for _, v := range args {
			v = utils.ExpandLocalPath(v)
			stat, err := os.Stat(v)
			if err != nil {
				fmt.Printf("Get file %s stat failed\n", v)
				logx.Error(err)
				continue
			}
			if stat.IsDir() {
				handleUploadFolder(&p, v)
			} else {
				handleUploadFile(&p, v)
			}
		}
	},
}

// Specifies the folder of the pikpak server
var uploadFolder string

// Specifies the file to upload
var uploadConcurrency int64

// Sync mode
var sync bool

// Parent path id
var parentId string

// Init upload command
func init() {
	UploadCmd.Flags().StringVarP(&uploadFolder, "path", "p", "/", "specific the folder of the pikpak server")
	UploadCmd.Flags().Int64VarP(&uploadConcurrency, "concurrency", "c", 1<<4, "specific the concurrency of the upload")
	UploadCmd.Flags().StringSliceVarP(&exclude, "exn", "e", []string{}, "specific the exclude file or folder")
	UploadCmd.Flags().BoolVarP(&sync, "sync", "s", false, "sync mode")
	UploadCmd.Flags().StringVarP(&parentId, "parent-id", "P", "", "parent folder id")
}

// Exclude string list
var exclude []string

var defaultExcludeRegexp []*regexp.Regexp = []*regexp.Regexp{
	// exclude the hidden file
	regexp.MustCompile(`^\..+`),
}

// Dispose the exclude file or folder
func disposeExclude() {
	for _, v := range exclude {
		defaultExcludeRegexp = append(defaultExcludeRegexp, regexp.MustCompile(v))
	}
}

func handleUploadFile(p *api.PikPak, path string) {
	var err error
	if parentId == "" {
		parentId, err = p.GetDeepFolderOrCreateId("", uploadFolder)
		if err != nil {
			fmt.Printf("Get folder %s id failed\n", uploadFolder)
			logx.Error(err)
			return
		}
	}
	err = p.UploadFile(parentId, path)
	if err != nil {
		fmt.Printf("Upload file %s failed\n", path)
		logx.Error(err)
		return
	}
	fmt.Printf("Upload file %s success!\n", path)
}

// upload files logic
func handleUploadFolder(p *api.PikPak, path string) {
	basePath := filepath.Base(filepath.ToSlash(path))
	uploadFilePath, err := utils.GetUploadFilePath(path, defaultExcludeRegexp)
	if err != nil {
		fmt.Println("Get upload file path failed")
		logx.Error(err)
		return
	}

	syncTxt, err := utils.NewSyncTxt(".pikpaksync.txt", sync)
	if err != nil {
		fmt.Println("Init sync file failed")
		logx.Error(err)
		return
	}
	defer syncTxt.Close()

	uploadFilePath = syncTxt.UnSync(uploadFilePath)

	fmt.Println("upload file list:")
	for _, f := range uploadFilePath {
		fmt.Println(filepath.Join(basePath, f))
	}

	if parentId == "" {
		parentId, err = p.GetDeepFolderOrCreateId("", uploadFolder)
		if err != nil {
			fmt.Printf("Get folder %s id error\n", uploadFolder)
			logx.Error(err)
			return
		}
	}

	logx.Debug("upload", "upload folder: ", uploadFolder, " parentId: ", parentId)

	parentId, err = p.GetDeepFolderOrCreateId(parentId, basePath)
	if err != nil {
		fmt.Printf("Get base_upload_path %s id error\n", basePath)
		logx.Error(err)
		return
	}
	parentIdMap := make(map[string]string)
	for _, v := range uploadFilePath {
		if strings.Contains(v, "/") || strings.Contains(v, "\\") {
			var id string
			base := filepath.Dir(v)

			// Avoid secondary query ids
			if mId, ok := parentIdMap[base]; !ok {
				id, err = p.GetDeepFolderOrCreateId(parentId, base)
				if err != nil {
					fmt.Println("Get folder id failed")
					logx.Error(err)
				}
				parentIdMap[base] = id
			} else {
				id = mId
			}

			err = p.UploadFile(id, filepath.Join(path, v))
			if err != nil {
				fmt.Printf("%s upload failed\n", v)
				logx.Error(err)
			}
			syncTxt.WriteString(v + "\n")
			fmt.Printf("%s upload success!\n", v)
		} else {
			err = p.UploadFile(parentId, filepath.Join(path, v))
			if err != nil {
				fmt.Printf("%s upload failed\n", v)
				logx.Error(err)
			}
			syncTxt.WriteString(v + "\n")
		}
	}
}
