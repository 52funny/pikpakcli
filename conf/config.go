package conf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v2"
)

var Config = struct {
	Proxy    string
	Username string
	Password string
}{}

var UseProxy = false

func InitConfig(path string) error {
	// first read the config info from executable file
	if readFromBinary() == nil {
		return nil
	}

	// read the config info from the config path
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	bs, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(bs, &Config)
	if err != nil {
		return err
	}

	// not empty
	// not contains '://'
	if len(Config.Proxy) != 0 && !strings.Contains(Config.Proxy, "://") {
		return fmt.Errorf("proxy should contains ://")
	} else if len(Config.Proxy) != 0 {
		UseProxy = true
	}
	return nil
}

// read config from binary in the end
// config_bytes: n bytes
// end_magic: 10 bytes
// size: 4 bytes
// -----------------------------------
// | config_bytes | size | end_magic |
// -----------------------------------
func readFromBinary() error {
	f, err := os.Open(os.Args[0])
	if err != nil {
		return err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return err
	}

	var end_magic = make([]byte, 10)
	n, err := f.ReadAt(end_magic, stat.Size()-10)
	if err != nil {
		return err
	}

	if n != 10 {
		return fmt.Errorf("read end_magic err: %d", n)
	}

	// not have `config.yml` in the end
	if !bytes.Equal(end_magic, []byte("config.yml")) {
		return fmt.Errorf("not a pikpakcli binary")
	}

	var size = make([]byte, 4)
	n, err = f.ReadAt(size, stat.Size()-14)

	if err != nil {
		return err
	}

	if n != 4 {
		return fmt.Errorf("read size err: %d", n)
	}

	configSize := int64(binary.LittleEndian.Uint32(size))
	configBuf := make([]byte, configSize)

	n, err = f.ReadAt(configBuf, stat.Size()-14-configSize)

	if err != nil || n != int(configSize) {
		return err
	}

	if n != int(configSize) {
		return fmt.Errorf("read config size err: %d", n)
	}

	// unmarshal config
	return yaml.Unmarshal(configBuf, &Config)
}
