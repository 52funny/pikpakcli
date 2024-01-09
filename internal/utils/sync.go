package utils

import (
	"errors"
	"io"
	"os"
	"slices"
	"strings"
	"unsafe"

	"github.com/sirupsen/logrus"
)

var ErrSyncTxtNotEnable = errors.New("sync txt is not enable")

type SyncTxt struct {
	Enable        bool
	FileName      string
	alreadySynced []string
	f             *os.File
}

func NewSyncTxt(fileName string, enable bool) (sync *SyncTxt, err error) {
	var f *os.File = nil
	var alreadySynced []string
	if enable {
		f, err = os.OpenFile(fileName, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
		if err != nil {
			return nil, err
		}
		bs, err := io.ReadAll(f)
		if err != nil {
			logrus.Error("read file error: ", err)
			os.Exit(1)
		}
		// avoid end with "\n"
		alreadySynced = strings.Split(
			strings.TrimRight(unsafe.String(unsafe.SliceData(bs), len(bs)), "\n"),
			"\n",
		)
	}
	return &SyncTxt{
		Enable:        enable,
		FileName:      fileName,
		f:             f,
		alreadySynced: alreadySynced,
	}, nil
}

// impl Writer
func (s *SyncTxt) Write(b []byte) (n int, err error) {
	if !s.Enable {
		return 0, ErrSyncTxtNotEnable
	}
	if b[len(b)-1] != '\n' {
		b = append(b, '\n')
	}
	// add to alreadySynced
	s.alreadySynced = append(s.alreadySynced, strings.TrimRight(string(b), "\n"))
	return s.f.Write(b)
}

// impl Closer
func (s *SyncTxt) Close() error {
	if !s.Enable {
		return ErrSyncTxtNotEnable
	}
	return s.f.Close()
}

// impl StringWriter
func (s *SyncTxt) WriteString(str string) (n int, err error) {
	if !s.Enable {
		return 0, ErrSyncTxtNotEnable
	}
	if str[len(str)-1] != '\n' {
		str += "\n"
	}
	// add to alreadySynced
	s.alreadySynced = append(s.alreadySynced, strings.TrimRight(str, "\n"))
	return s.f.WriteString(str)
}

func (s *SyncTxt) UnSync(files []string) []string {
	if !s.Enable {
		return files
	}
	res := make([]string, 0)
	for _, f := range files {
		if slices.Contains(s.alreadySynced, f) {
			continue
		}
		res = append(res, f)
	}
	return res
}
