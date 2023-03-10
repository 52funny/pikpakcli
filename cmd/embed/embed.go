package embed

import (
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

const magic = "config.yml"

var EmbedCmd = &cobra.Command{
	Use:   "embed",
	Short: `Embed config file`,
	Run: func(cmd *cobra.Command, args []string) {
		// fmt.Println(args)
		if len(args) <= 0 {
			logrus.Errorln("Please specify the config file path")
			os.Exit(1)
		}
		ok, err := checkEmbed()
		if err != nil {
			logrus.Errorln("check magic error", err)
			os.Exit(1)
		}
		if ok {
			logrus.Infoln("config file has been embedded")
			os.Exit(0)
		}
		err = embed(args)
		if err != nil {
			logrus.Errorln(err)
			os.Exit(1)
		}
		logrus.Infoln("Embed config file success")
	},
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
func embed(args []string) error {
	f, err := os.Open(args[0])
	if err != nil {
		return fmt.Errorf("open config file error: %s", err.Error())
	}
	defer f.Close()
	fStat, _ := f.Stat()
	bs, err := ioutil.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read config file error: %s", err.Error())
	}
	sizeBytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(sizeBytes, uint32(fStat.Size()))
	bs = append(bs, sizeBytes...)
	bs = append(bs, []byte("config.yml")...)
	binFile, err := os.OpenFile(os.Args[0], os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("open binary file error: %s", err.Error())
	}
	defer binFile.Close()
	n, err := binFile.Write(bs)
	if err != nil || n != len(bs) {
		return fmt.Errorf("write to binary error: %s", err.Error())
	}
	return nil
}
