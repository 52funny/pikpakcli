package pikpak

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
	"github.com/vbauerster/mpb/v8"
)

// Download file
func (f *File) Download(path string, bar *mpb.Bar) error {
	outFile, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()
	info, err := outFile.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	resume := size != 0
	req, err := http.NewRequest("GET", f.Links.ApplicationOctetStream.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	if resume {
		if bar != nil {
			bar.SetCurrent(size)
		}
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", size))
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	// defer resp.Body.Close()
	if resume && resp.StatusCode != http.StatusPartialContent {
		logrus.Warnf("Resume file %s failed: Server doesn't support, restarting from the beginning", f.Name)
		// try re-opening the file with contents truncated
		outFile.Close()
		outFile, err = os.Create(path)
		if err != nil {
			return err
		}
	}
	contentLength, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return errors.New("transmute content length to int64 failed")
	}
	var proxyReader io.ReadCloser
	if bar != nil {
		proxyReader = bar.ProxyReader(resp.Body)
	} else {
		proxyReader = resp.Body
	}
	defer proxyReader.Close()

	// using 128K buffer to accelerate the download
	buf := make([]byte, 1024*128)

	written, err := io.CopyBuffer(outFile, proxyReader, buf)
	if err != nil {
		return err
	}
	if contentLength != written {
		return errors.New("content length not equal to written")
	}
	return nil
}
