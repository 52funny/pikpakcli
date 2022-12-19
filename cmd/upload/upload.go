package upload

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var UploadCmd = &cobra.Command{
	Use:     "upload",
	Aliases: []string{"u"},
	Short:   `Upload file to pikpak server`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) > 0 {
			handleUpload(args[0])
		} else {
			logrus.Errorln("Please specific a file to upload")
		}
	},
}

// Specifies the folder of the pikpak server
var uploadFolder string

// Specifies the file to upload
var uploadConcurrency int64

// Sync mode
var sync bool

// Init upload command
func init() {
	UploadCmd.Flags().StringVarP(&uploadFolder, "path", "p", "", "specific the folder of the pikpak server")
	UploadCmd.Flags().Int64VarP(&uploadConcurrency, "concurrency", "c", 1<<4, "specific the concurrency of the upload")
	UploadCmd.Flags().StringSliceVarP(&exclude, "exn", "e", []string{}, "specific the exclude file or folder")
	UploadCmd.Flags().BoolVarP(&sync, "sync", "s", false, "sync mode")
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

// upload files logic
func handleUpload(path string) {
	pikpak.Concurrent = uploadConcurrency
	uploadFilePath := utils.GetUploadFilePath(path, defaultExcludeRegexp)

	var f *os.File

	var parentId string

	// sync mode
	if sync {
		file, err := os.OpenFile(filepath.Join(".", ".pikpaksync.txt"), os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			logrus.Error(err)
			os.Exit(1)
		}
		f = file
		bs, err := ioutil.ReadAll(f)
		if err != nil {
			logrus.Error("read file error: ", err)
			os.Exit(1)
		}
		alreadySyncFiles := strings.Split(string(bs), "\n")
		files := make([]string, 0)
		for _, f := range uploadFilePath {
			if !utils.Contains(alreadySyncFiles, f) {
				files = append(files, f)
			}
		}
		uploadFilePath = files
	}

	logrus.Info("upload file list:")
	for _, f := range uploadFilePath {
		logrus.Infoln(f)
	}

	p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)

	err := p.Login()
	if err != nil {
		logrus.Error(err)
	}
	err = p.AuthCaptchaToken("POST:/drive/v1/files")
	if err != nil {
		logrus.Error(err)
	}

	go func() {
		ticker := time.NewTicker(time.Second * 7200 * 3 / 4)
		defer ticker.Stop()
		for range ticker.C {
			err := p.RefreshToken()
			if err != nil {
				logrus.Warn(err)
				continue
			}
		}
	}()

	if uploadFolder != "" {
		parentPathS := strings.Split(uploadFolder, "/")
		for i, v := range parentPathS {
			if v == "." {
				parentPathS = append(parentPathS[:i], parentPathS[i+1:]...)
			}
		}
		id, err := p.GetDeepParentOrCreateId(parentId, parentPathS)
		if err != nil {
			logrus.Error(err)
			os.Exit(-1)
		} else {
			parentId = id
		}
	}
	logrus.Debug("upload folder: ", uploadFolder, " parentId: ", parentId)

	for _, v := range uploadFilePath {
		if strings.Contains(v, "/") {
			basePath := filepath.Dir(v)
			basePathS := strings.Split(basePath, "/")
			id, err := p.GetDeepParentOrCreateId(parentId, basePathS)
			if err != nil {
				logrus.Error(err)
			}
			err = p.UploadFile(id, filepath.Join(path, v))
			if err != nil {
				logrus.Error(err)
			}
			if sync {
				f.WriteString(v + "\n")
			}
			logrus.Infof("%s upload completed!\n", v)
		} else {
			err = p.UploadFile(parentId, filepath.Join(path, v))
			if err != nil {
				logrus.Error(err)
			}
			if sync {
				f.WriteString(v + "\n")
			}
		}
	}
}
