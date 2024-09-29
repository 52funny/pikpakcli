package download

import (
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/52funny/fastdown"
	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

// Number of simultaneous downloads
//
// default 1
var count int

// Number of go routines are used to download each file
var maxRoutine int = 1

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

type warpFile struct {
	f      *pikpak.File
	output string
}

type warpStat struct {
	s      pikpak.FileStat
	output string
}

func init() {
	DownloadCmd.Flags().IntVarP(&count, "count", "c", 1, "number of simultaneous downloads")
	DownloadCmd.Flags().StringVarP(&output, "output", "o", ".", "output directory")
	DownloadCmd.Flags().StringVarP(&folder, "path", "p", "/", "specific the folder of the pikpak server\nonly support download folder")
	DownloadCmd.Flags().StringVarP(&parentId, "parent-id", "P", "", "the parent path id")
}

var DownloadCmd = &cobra.Command{
	Use:     "download",
	Aliases: []string{"d"},
	Short:   `Download file from pikpak server`,
	Run: func(cmd *cobra.Command, args []string) {
		p := pikpak.NewPikPak(conf.Config.Username, conf.Config.Password)
		err := p.Login()
		if err != nil {
			logrus.Errorln("Login Failed:", err)
		}
		var collectStat []warpStat
		if len(args) > 0 {
			cst, err := collectFileStat(&p, args)
			if err != nil {
				logrus.Errorln("Collect File Stat Failed:", err)
				return
			}
			collectStat = cst
		} else {
			cst, err := collectFolderStat(&p)
			if err != nil {
				logrus.Errorln("Collect Folder Stat Failed:", err)
				return
			}
			collectStat = cst
		}

		for _, st := range collectStat {
			logrus.Infoln("Download:", st.output, st.s.Name)
		}
		in := make(chan warpFile, count)
		wait := new(sync.WaitGroup)

		for i := 0; i < count; i++ {
			wait.Add(1)
			go download(in, wait)
		}
		for _, st := range collectStat {
			f, err := p.GetFile(st.s.ID)
			if err != nil {
				logrus.Errorln("Get File Failed:", err)
				continue
			}
			in <- warpFile{
				f:      &f,
				output: st.output,
			}
		}
		close(in)
		wait.Wait()
	},
}

// Downloads all files in the specified directory
func collectFolderStat(p *pikpak.PikPak) ([]warpStat, error) {
	base := filepath.Base(folder)
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(folder)
		if err != nil {
			return nil, err
		}
	}
	collectStat := make([]warpStat, 0)
	recursive(p, &collectStat, parentId, filepath.Join(output, base))
	return collectStat, nil
}

func recursive(p *pikpak.PikPak, collectWarpFile *[]warpStat, parentId string, parentPath string) {
	statList, err := p.GetFolderFileStatList(parentId)
	if err != nil {
		logrus.Errorln("Get Folder File Stat List Failed:", err)
		return
	}
	for _, r := range statList {
		if r.Kind == "drive#folder" {
			recursive(p, collectWarpFile, r.ID, filepath.Join(parentPath, r.Name))
		} else {
			*collectWarpFile = append(*collectWarpFile, warpStat{
				s:      r,
				output: parentPath,
			})
		}
	}
}

func collectFileStat(p *pikpak.PikPak, args []string) ([]warpStat, error) {
	var err error
	if parentId == "" {
		parentId, err = p.GetPathFolderId(folder)
		if err != nil {
			return nil, err
		}
	}

	collectStat := make([]warpStat, 0, len(args))

	for _, path := range args {
		stat, err := p.GetFileStat(parentId, path)
		if err != nil {
			logrus.Errorln(path, "get parent id failed:", err)
			continue
		}
		collectStat = append(collectStat, warpStat{
			s:      stat,
			output: output,
		})
	}
	return collectStat, nil
}

func download(inCh <-chan warpFile, wait *sync.WaitGroup) {
	defer wait.Done()
	for warp := range inCh {
		// fmt.Println("in", warp, warp.f.Links.ApplicationOctetStream.URL)
		utils.CreateDirIfNotExist(warp.output)

		path := filepath.Join(warp.output, warp.f.Name)
		exist, _ := utils.Exists(path)
		if exist {
			st, err := os.Stat(path)
			if err != nil {
				continue
			}
			remoteSize, _ := strconv.ParseInt(warp.f.Size, 10, 64)

			// if the file size is the same, skip downloading
			if st.Size() == remoteSize {
				continue
			}
		}

		sz, err := strconv.ParseInt(warp.f.Size, 10, 64)
		if err != nil {
			logrus.Errorln("ParseInt", warp.f.Size, "failed", err)
			continue
		}
		dw := fastdown.NewDownloadWrapper(warp.f.Links.ApplicationOctetStream.URL, maxRoutine, sz, warp.output, warp.f.Name)
		err = dw.Download()
		if err != nil {
			logrus.Errorln("Download", warp.f.Name, "failed", err)
			continue
		}
	}
}
