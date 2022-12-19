package conf

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

var Config = struct {
	Username string
	Password string
}{}

func InitConfig(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	bs, err := ioutil.ReadAll(f)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(bs, &Config)
}
