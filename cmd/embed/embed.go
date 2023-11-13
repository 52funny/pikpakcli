package embed

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const magic = "config.yml"

var EmbedCmd = &cobra.Command{
	Use:   "embed",
	Short: `Embed config file`,
	Run: func(cmd *cobra.Command, args []string) {
		if len(args) <= 0 {
			logrus.Errorln("Please specify the config file path")
			os.Exit(1)
		}
		ok, err := checkEmbed()
		if err != nil {
			logrus.Errorln("check magic error", err)
			os.Exit(1)
		}

		if update && ok {
			err = updateEmbed(args[0], os.Args[0])
			if err != nil {
				logrus.Errorln(err)
				os.Exit(1)
			}
			logrus.Infoln("Update embed config file success")
			os.Exit(0)
		}

		if ok {
			logrus.Warnln("config file has been embedded")
			os.Exit(1)
		}
		copyBin, err := copyBin(os.Args[0])
		if err != nil {
			logrus.Errorln("create copy binary file error:", err.Error())
			os.Exit(1)
		}
		err = embed(args[0], copyBin)
		if err != nil {
			logrus.Errorln(err)
			os.Exit(1)
		}
		logrus.Infoln("Embed config file success")
	},
}

var update bool

func init() {
	EmbedCmd.Flags().BoolVarP(&update, "update", "u", false, "update embed config")
}

func checkEmbed() (bool, error) {
	f, err := os.Open(os.Args[0])
	if err != nil {
		return false, err
	}
	defer f.Close()
	fStat, _ := f.Stat()
	magicBuf := make([]byte, len(magic))
	n, err := f.ReadAt(magicBuf, fStat.Size()-int64(len(magic)))
	if err != nil {
		return false, err
	}
	if n != len(magic) {
		return false, fmt.Errorf("read magic size error")
	}
	return string(magicBuf) == magic, nil
}

// embed config file to binary
func embed(configPath string, binFile *os.File) error {
	f, err := os.Open(configPath)
	if err != nil {
		return fmt.Errorf("open config file error: %s", err.Error())
	}
	defer f.Close()
	fStat, _ := f.Stat()
	bs, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read config file error: %s", err.Error())
	}
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, uint32(fStat.Size()))
	bs = append(bs, sizeBytes...)
	bs = append(bs, []byte("config.yml")...)
	// binFile, err := os.OpenFile(binPath, os.O_WRONLY|os.O_APPEND, 0666)
	// if err != nil {
	// 	return fmt.Errorf("open binary file error: %s", err.Error())
	// }
	defer binFile.Close()
	n, err := binFile.Write(bs)
	if err != nil || n != len(bs) {
		return fmt.Errorf("write to binary error: %s", err.Error())
	}
	return nil
}

// first remove embed config
// second embed config
func updateEmbed(configPath string, BinPath string) error {
	copyBin, err := copyBin(BinPath)
	if err != nil {
		return err
	}
	binStat, _ := copyBin.Stat()
	var size = make([]byte, 4)
	n, err := copyBin.ReadAt(size, binStat.Size()-14)

	if err != nil {
		return err
	}

	if n != 4 {
		return fmt.Errorf("read size err: %d", n)
	}

	configSize := int64(binary.LittleEndian.Uint32(size))
	copyBin.Seek(binStat.Size()-14-configSize, 0)
	err = copyBin.Truncate(binStat.Size() - 14 - configSize)
	if err != nil {
		return err
	}

	return embed(configPath, copyBin)
}

func copyBin(path string) (*os.File, error) {
	binFile, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open binary file error: %s", err.Error())
	}
	copyBinName := utils.GetEmbedBinName(os.Args[0])
	binSt, _ := binFile.Stat()
	copyBin, err := os.OpenFile(copyBinName, os.O_RDWR|os.O_CREATE|os.O_TRUNC, binSt.Mode())
	if err != nil {
		return nil, fmt.Errorf("open copy binary file error: %s", err.Error())
	}
	io.Copy(copyBin, binFile)
	binFile.Close()
	return copyBin, nil
}

// delete some bytes in the end of file
func deleteBytes(f *os.File, n int64) error {
	fStat, _ := f.Stat()
	// read the last n bytes
	bs := make([]byte, n)
	_, err := f.ReadAt(bs, fStat.Size()-n)
	if err != nil {
		return err
	}
	// truncate file
	err = f.Truncate(fStat.Size() - n)
	if err != nil {
		return err
	}
	return nil
}
