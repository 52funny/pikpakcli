package utils

import (
	"os"
	"path/filepath"
	"regexp"

	"github.com/sirupsen/logrus"
)

func Contains(alreadySyncFiles []string, f string) bool {
	for _, v := range alreadySyncFiles {
		if v == f {
			return true
		}
	}
	return false
}

// 获取目录文件夹下的所有文件路径名
func GetUploadFilePath(basePath string, defaultRegexp []*regexp.Regexp) []string {
	rawPath := make([]string, 0)
	state, err := os.Stat(basePath)
	if err != nil {
		logrus.Error(err)
		return nil
	}
	if state.IsDir() {
		filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logrus.Error(err)
				return nil
			}
			for _, r := range defaultRegexp {
				if r.MatchString(info.Name()) {
					if info.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			if info.IsDir() {
				return nil
			}
			p, _ := filepath.Rel(basePath, path)
			rawPath = append(rawPath, p)
			return nil
		})
	} else {
		for _, r := range defaultRegexp {
			if r.MatchString(state.Name()) {
				return nil
			}
		}
		// index := strings.Index(path, base)
		// if index > 0 {
		// 	rawPath = append(rawPath, path[index+len(base):])
		// }
		p, _ := filepath.Rel(basePath, basePath)
		rawPath = append(rawPath, p)
	}
	return rawPath
}

// 不存在目录就创建
func CreateDirIfNotExist(path string) error {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(path, os.ModePerm)
			if err != nil {
				return err
			}
		} else {
			return err
		}
	}
	return nil
}
