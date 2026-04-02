package api

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDownloadResumesAfterInterruptedTransfer(t *testing.T) {
	content := []byte("hello world")
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch requests.Add(1) {
		case 1:
			require.Empty(t, r.Header.Get("Range"))
			hj, ok := w.(http.Hijacker)
			require.True(t, ok)
			conn, rw, err := hj.Hijack()
			require.NoError(t, err)
			defer conn.Close()

			_, err = fmt.Fprintf(rw, "HTTP/1.1 200 OK\r\nContent-Length: %d\r\n\r\n", len(content))
			require.NoError(t, err)
			_, err = rw.Write(content[:5])
			require.NoError(t, err)
			require.NoError(t, rw.Flush())
		case 2:
			require.Equal(t, "bytes=5-", r.Header.Get("Range"))
			remaining := content[5:]
			w.Header().Set("Content-Length", strconv.Itoa(len(remaining)))
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 5-%d/%d", len(content)-1, len(content)))
			w.WriteHeader(http.StatusPartialContent)
			_, err := w.Write(remaining)
			require.NoError(t, err)
		default:
			t.Fatalf("unexpected request count: %d", requests.Load())
		}
	}))
	defer server.Close()

	file := File{
		FileStat: FileStat{
			Name: "resume.bin",
			Size: strconv.Itoa(len(content)),
		},
	}
	file.Links.ApplicationOctetStream.URL = server.URL

	target := filepath.Join(t.TempDir(), file.Name)
	require.NoError(t, file.Download(target, nil))

	downloaded, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, content, downloaded)
	require.EqualValues(t, 2, requests.Load())
}

func TestDownloadRestartsWhenServerIgnoresRangeRequest(t *testing.T) {
	content := []byte("hello world")
	var requests atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch requests.Add(1) {
		case 1:
			require.Equal(t, "bytes=5-", r.Header.Get("Range"))
		case 2:
			require.Empty(t, r.Header.Get("Range"))
		default:
			t.Fatalf("unexpected request count: %d", requests.Load())
		}

		w.Header().Set("Content-Length", strconv.Itoa(len(content)))
		w.WriteHeader(http.StatusOK)
		_, err := w.Write(content)
		require.NoError(t, err)
	}))
	defer server.Close()

	file := File{
		FileStat: FileStat{
			Name: "restart.bin",
			Size: strconv.Itoa(len(content)),
		},
	}
	file.Links.ApplicationOctetStream.URL = server.URL

	target := filepath.Join(t.TempDir(), file.Name)
	require.NoError(t, os.WriteFile(target, content[:5], 0644))

	require.NoError(t, file.Download(target, nil))

	downloaded, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, content, downloaded)
	require.EqualValues(t, 2, requests.Load())
}

func TestDownloadTreatsSatisfiedRangeAsSuccess(t *testing.T) {
	content := []byte("hello world")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "bytes=11-", r.Header.Get("Range"))
		w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
	}))
	defer server.Close()

	file := File{
		FileStat: FileStat{
			Name: "complete.bin",
			Size: strconv.Itoa(len(content)),
		},
	}
	file.Links.ApplicationOctetStream.URL = server.URL

	target := filepath.Join(t.TempDir(), file.Name)
	require.NoError(t, os.WriteFile(target, content, 0644))

	require.NoError(t, file.Download(target, nil))

	f, err := os.Open(target)
	require.NoError(t, err)
	defer f.Close()

	reader := bufio.NewReader(f)
	got, err := reader.Peek(len(content))
	require.NoError(t, err)
	require.Equal(t, content, got)
}
