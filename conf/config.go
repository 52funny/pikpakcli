package conf

import (
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
	// Firstly, it reads config.yml from the given path.
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
