package download

import (
	"path/filepath"

	"github.com/52funny/pikpakcli/conf"
	"github.com/52funny/pikpakcli/internal/pikpak"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

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
		if len(args) > 0 {
			downloadFile(&p, args)
		} else {
			downloadFolder(&p)
		}
	},
}

// Number of simultaneous downloads
//
// default 3
var count int

// Specifies the folder of the pikpak server
//
// default server root directory (.)
var folder string

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
	DownloadCmd.Flags().IntVarP(&count, "count", "c", 3, "number of simultaneous downloads")
	DownloadCmd.Flags().StringVarP(&output, "output", "o", "", "output directory")
	DownloadCmd.Flags().StringVarP(&folder, "path", "p", "/", "specific the folder of the pikpak server\nonly support download folder")
}

// Downloads all files in the specified directory
func downloadFolder(p *pikpak.PikPak) {
	base := filepath.Base(folder)
	parentId, err := p.GetPathFolderId(folder)
	if err != nil {
		logrus.Errorln("Get Parent Folder Id Failed:", err)
		return
	}
	collectStat := make([]warpStat, 0)
	recursive(p, &collectStat, parentId, filepath.Join(output, base))

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
					logrus.Errorln("Get File Failed:", err)
				}
				fileCh <- warpFile{
					f:      &file,
					output: stat.output,
				}
				statDone <- struct{}{}
			}
		}(fileCh, statCh, statDone)
	}

	for i := 0; i < count; i++ {
		go download(fileCh, fileDone)
	}

	for i := 0; i < len(collectStat); i += 1 {
		err := utils.CreateDirIfNotExist(collectStat[i].output)
		if err != nil {
			logrus.Errorln("Create output directory failed:", err)
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
			// file, _ := p.GetFile(r.ID)
			*collectWarpFile = append(*collectWarpFile, warpStat{
				s:      r,
				output: parentPath,
			})
			// fmt.Println(r.Name, r.Size, r.Kind, parentPath)
		}
	}
}

func downloadFile(p *pikpak.PikPak, args []string) {
	files := make([]string, 0, len(args))
	for _, v := range args {
		files = append(files, filepath.Join(folder, v))
	}

	sendCh := make(chan warpFile, 1)
	receiveCh := make(chan struct{}, len(files))
	for i := 0; i < count; i++ {
		go download(sendCh, receiveCh)
	}
	for _, f := range files {
		dir, base := filepath.Dir(f), filepath.Base(f)
		id, err := p.GetPathFolderId(dir)
		if err != nil {
			logrus.Errorln(dir, "Get Parent Folder Id Failed:", err)
			continue
		}
		stat, err := p.GetFileStat(id, base)
		if err != nil {
			logrus.Errorln(base, "Get File Stat Failed:", err)
			continue
		}
		file, err := p.GetFile(stat.ID)
		if err != nil {
			logrus.Errorln(base, "Get File Failed:", err)
			continue
		}
		sendCh <- warpFile{
			f:      &file,
			output: output,
		}
	}
	close(sendCh)
	for i := 0; i < len(files); i++ {
		<-receiveCh
	}
	close(receiveCh)

}

func download(inCh <-chan warpFile, out chan<- struct{}) {
	for {
		warp, ok := <-inCh
		if !ok {
			break
		}
		err := warp.f.Download(warp.output)
		if err != nil {
			logrus.Errorln("Download", warp.f.Name, "Failed:", err)
		} else {
			logrus.Infoln("Download", warp.f.Name, "Success")
		}
		out <- struct{}{}
	}
}
