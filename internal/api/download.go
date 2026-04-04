package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strconv"

	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/vbauerster/mpb/v8"
)

const maxDownloadRetries = 3

var errRestartDownload = errors.New("restart download from beginning")

type retryableDownloadError struct {
	err error
}

func (f *File) requestContext() context.Context {
	if f != nil && f.ctx != nil {
		return f.ctx
	}
	return context.Background()
}

func (e *retryableDownloadError) Error() string {
	return e.err.Error()
}

func (e *retryableDownloadError) Unwrap() error {
	return e.err
}

func retryableDownload(err error) error {
	if err == nil {
		return nil
	}
	return &retryableDownloadError{err: err}
}

func isRetryableDownloadError(err error) bool {
	var target *retryableDownloadError
	return errors.As(err, &target)
}

// Download file
func (f *File) Download(path string, bar *mpb.Bar) error {
	expectedSize, err := strconv.ParseInt(f.Size, 10, 64)
	if err != nil {
		expectedSize = -1
	}

	var lastErr error
	for attempt := 0; attempt < maxDownloadRetries; attempt++ {
		lastErr = f.download(path, bar, expectedSize)
		if lastErr == nil {
			return nil
		}
		if !isRetryableDownloadError(lastErr) {
			return lastErr
		}
		if attempt == maxDownloadRetries-1 {
			break
		}
		logx.Warnf("transfer", "Download %s interrupted, retrying (%d/%d): %v", f.Name, attempt+1, maxDownloadRetries-1, lastErr)
	}

	return lastErr
}

func (f *File) download(path string, bar *mpb.Bar, expectedSize int64) error {
	outFile, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer outFile.Close()

	info, err := outFile.Stat()
	if err != nil {
		return err
	}
	offset := info.Size()

	if expectedSize >= 0 && offset > expectedSize {
		if err := outFile.Truncate(0); err != nil {
			return err
		}
		offset = 0
	}

	if _, err := outFile.Seek(offset, io.SeekStart); err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(f.requestContext(), "GET", f.Links.ApplicationOctetStream.URL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", userAgent)
	if offset > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", offset))
		if bar != nil {
			bar.SetCurrent(offset)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return retryableDownload(err)
	}
	defer resp.Body.Close()

	switch {
	case offset > 0 && resp.StatusCode == http.StatusRequestedRangeNotSatisfiable:
		if expectedSize >= 0 && offset == expectedSize {
			if bar != nil {
				bar.SetCurrent(expectedSize)
			}
			return nil
		}
		if err := outFile.Truncate(0); err != nil {
			return err
		}
		if bar != nil {
			bar.SetCurrent(0)
		}
		return retryableDownload(errRestartDownload)
	case offset > 0 && resp.StatusCode == http.StatusOK:
		logx.Warnf("transfer", "Resume file %s failed: server ignored range request, restarting from the beginning", f.Name)
		if err := outFile.Truncate(0); err != nil {
			return err
		}
		if bar != nil {
			bar.SetCurrent(0)
		}
		return retryableDownload(errRestartDownload)
	case offset > 0 && resp.StatusCode != http.StatusPartialContent:
		if resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests {
			return retryableDownload(fmt.Errorf("download request failed: %s", resp.Status))
		}
		return fmt.Errorf("download request failed: %s", resp.Status)
	case offset == 0 && (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices):
		if resp.StatusCode >= http.StatusInternalServerError || resp.StatusCode == http.StatusTooManyRequests {
			return retryableDownload(fmt.Errorf("download request failed: %s", resp.Status))
		}
		return fmt.Errorf("download request failed: %s", resp.Status)
	}

	var reader io.ReadCloser
	if bar != nil {
		reader = bar.ProxyReader(resp.Body)
	} else {
		reader = resp.Body
	}
	defer reader.Close()

	buf := make([]byte, 1024*128)
	written, err := io.CopyBuffer(outFile, reader, buf)
	if err != nil {
		var netErr net.Error
		if errors.Is(err, io.ErrUnexpectedEOF) || errors.As(err, &netErr) {
			return retryableDownload(err)
		}
		return retryableDownload(err)
	}

	contentLengthHeader := resp.Header.Get("Content-Length")
	if contentLengthHeader != "" {
		contentLength, err := strconv.ParseInt(contentLengthHeader, 10, 64)
		if err != nil {
			return fmt.Errorf("parse content length failed: %w", err)
		}
		if contentLength != written {
			return retryableDownload(fmt.Errorf("content length not equal to written"))
		}
	}

	if expectedSize >= 0 && offset+written != expectedSize {
		return retryableDownload(fmt.Errorf("download incomplete: got %d of %d bytes", offset+written, expectedSize))
	}

	return nil
}
