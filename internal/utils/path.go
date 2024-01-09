package utils

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

func Contains(alreadySyncFiles []string, f string) bool {
	return slices.Contains(alreadySyncFiles, f)
}

func SplitSeparator(path string) []string {
	if path == "" {
		return []string{}
	}
	return strings.Split(path, string(filepath.Separator))
}

func Slash(path string) string {
	// clean path
	path = filepath.Clean(path)
	if path == "" {
		return ""
	}
	if path[0] == filepath.Separator {
		return path[1:]
	}
	return path
}

// 获取目录文件夹下的所有文件路径名
func GetUploadFilePath(basePath string, defaultRegexp []*regexp.Regexp) ([]string, error) {
	rawPath := make([]string, 0)
	err := filepath.WalkDir(basePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// match regexp
		// if matched, then skip
		// else append
		matchRegexp := func(name string) bool {
			for _, r := range defaultRegexp {
				if r.MatchString(name) {
					return true
				}
			}
			return false
		}

		if matchRegexp(d.Name()) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// skip dir
		if d.IsDir() {
			return nil
		}
		// get relative path
		refPath, err := filepath.Rel(basePath, path)
		if err != nil {
			return err
		}
		// append to rawPath
		rawPath = append(rawPath, refPath)
		return nil
	})
	return rawPath, err
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
