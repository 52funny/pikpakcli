package pikpak

import (
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
)

// Download file
func (f *File) Download(output string) error {
	req, err := http.NewRequest("GET", f.Links.ApplicationOctetStream.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	newFile, err := os.Create(filepath.Join(output, f.Name))
	if err != nil {
		return err
	}
	contentLength, err := strconv.ParseInt(resp.Header.Get("Content-Length"), 10, 64)
	if err != nil {
		return errors.New("transmute content length to int64 failed")
	}
	written, err := io.Copy(newFile, resp.Body)
	if err != nil {
		return err
	}
	if contentLength != written {
		return errors.New("content length not equal to written")
	}
	return nil
}
