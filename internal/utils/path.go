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

func Slash(path string) string {
	path = filepath.Clean(path)
	if path == "" {
		return ""
	}
	if path[0] == '/' {
		return path[1:]
	}
	return path
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

// 检查路径是否存在
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// 不存在目录就创建
func CreateDirIfNotExist(path string) error {
	exist, err := Exists(path)
	if err != nil {
		return err
	}
	if !exist {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	return nil
}

// 创建空文件
func TouchFile(path string) error {
	exist, err := Exists(path)
	if err != nil {
		return err
	}
	if !exist {
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		f.Close()
	}
	return nil
}
