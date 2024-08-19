package conf

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v2"
)

type ConfigType struct {
	Proxy    string
	Username string
	Password string
}

var Config ConfigType

// UseProxy returns whether the proxy is used
func (c *ConfigType) UseProxy() bool {
	return len(c.Proxy) != 0
}

// Initializing configuration information
func InitConfig(path string) error {
	// Firstly, read the config info from executable file
	if readFromBinary() == nil {
		return nil
	}

	// Secondly, it reads config.yml from the given path.
	// If there is no config.yml in the given path, it reads it from the default config path.
	_, err := os.Stat(path)
	switch os.IsNotExist(err) {
	case true:
		if err := readFromConfigDir(); err != nil {
			return err
		}
	case false:
		if err := readFromPath(path); err != nil {
			return err
		}
	}

	// Not empty
	// Must contains '://'
	if len(Config.Proxy) != 0 && !strings.Contains(Config.Proxy, "://") {
		return fmt.Errorf("proxy should contains ://")
	}
	return nil
}

// Read config from binary in the end
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

	// Not have `config.yml` in the end
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

	// Unmarshal config
	return yaml.Unmarshal(configBuf, &Config)
}

// Read configuration file from the given path
func readFromPath(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	bs, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(bs, &Config)
	if err != nil {
		return err
	}
	return nil
}

// Read configuration file from config path
func readFromConfigDir() error {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return err
	}
	path := filepath.Join(configDir, "pikpakcli", "config.yml")
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	bs, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(bs, &Config)
	if err != nil {
		return err
	}
	return nil
}
