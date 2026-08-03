package api

import (
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestShareInfoCollectsFileIDsViaDetailPagination(t *testing.T) {
	var shareQueries, detailQueries []url.Values
	p := &PikPak{DeviceId: "device-id", JwtToken: "jwt-token"}
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "device-id", req.Header.Get("X-Peer-Id"))
		switch req.URL.Path {
		case "/drive/v1/share":
			shareQueries = append(shareQueries, req.URL.Query())
			return testHTTPResponse(http.StatusOK, `{
				"share_status":"OK",
				"pass_code_token":"token-1",
				"title":"My Share"
			}`), nil
		case "/drive/v1/share/detail":
			detailQueries = append(detailQueries, req.URL.Query())
			if req.URL.Query().Get("page_token") == "p2" {
				return testHTTPResponse(http.StatusOK, `{
					"share_status":"OK",
					"files":[{"id":"f3","name":"c.txt"}]
				}`), nil
			}
			return testHTTPResponse(http.StatusOK, `{
				"share_status":"OK",
				"next_page_token":"p2",
				"files":[{"id":"f1","name":"a.txt"},{"id":"f2","name":"b.txt"}]
			}`), nil
		default:
			t.Fatalf("unexpected path %s", req.URL.Path)
			return nil, nil
		}
	})}

	token, title, fileIDs, err := p.ShareInfo("abc123", "1234")

	require.NoError(t, err)
	require.Equal(t, "token-1", token)
	require.Equal(t, "My Share", title)
	require.Equal(t, []string{"f1", "f2", "f3"}, fileIDs)

	require.Len(t, shareQueries, 1)
	require.Equal(t, "100", shareQueries[0].Get("limit"))
	require.Equal(t, "abc123", shareQueries[0].Get("share_id"))
	require.Equal(t, "1234", shareQueries[0].Get("pass_code"))

	require.Len(t, detailQueries, 2)
	require.Equal(t, "token-1", detailQueries[0].Get("pass_code_token"))
	require.Equal(t, "", detailQueries[0].Get("page_token"))
	require.Equal(t, "p2", detailQueries[1].Get("page_token"))
}

func TestShareInfoRejectsDuplicatePageToken(t *testing.T) {
	var detailRequests int
	p := &PikPak{}
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/drive/v1/share":
			return testHTTPResponse(http.StatusOK, `{"share_status":"OK","pass_code_token":"token-1"}`), nil
		case "/drive/v1/share/detail":
			detailRequests++
			return testHTTPResponse(http.StatusOK, `{"share_status":"OK","next_page_token":"stuck","files":[{"id":"f1"}]}`), nil
		default:
			t.Fatalf("unexpected path %s", req.URL.Path)
			return nil, nil
		}
	})}

	_, _, _, err := p.ShareInfo("abc123", "")

	require.ErrorContains(t, err, "pagination stuck on next_page_token")
	require.Equal(t, 2, detailRequests)
}

func TestShareInfoOmitPassCodeWhenEmpty(t *testing.T) {
	var gotPassCode string
	p := &PikPak{}
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Path {
		case "/drive/v1/share":
			gotPassCode = req.URL.Query().Get("pass_code")
			return testHTTPResponse(http.StatusOK, `{"share_status":"OK","pass_code_token":"token-1"}`), nil
		case "/drive/v1/share/detail":
			return testHTTPResponse(http.StatusOK, `{"share_status":"OK","files":[]}`), nil
		default:
			t.Fatalf("unexpected path %s", req.URL.Path)
			return nil, nil
		}
	})}

	_, _, _, err := p.ShareInfo("abc123", "")

	require.NoError(t, err)
	require.Equal(t, "", gotPassCode)
}

func TestShareInfoStructuredError(t *testing.T) {
	p := &PikPak{}
	p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return testHTTPResponse(http.StatusOK, `{"error_code":3,"error":"invalid_argument","error_description":"Request parameter error"}`), nil
	})}

	_, _, _, err := p.ShareInfo("abc123", "")

	require.EqualError(t, err, "share access failed: Request parameter error")
}

func TestShareInfoHTTPError(t *testing.T) {
	p := &PikPak{}
	p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return testHTTPResponse(http.StatusBadGateway, `<html>bad gateway</html>`), nil
	})}

	_, _, _, err := p.ShareInfo("abc123", "")

	require.ErrorContains(t, err, "HTTP 502 Bad Gateway")
}

func TestShareInfoRejectsFailedShareStatus(t *testing.T) {
	p := &PikPak{}
	p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return testHTTPResponse(http.StatusOK, `{"share_status":"NOT_FOUND","share_status_text":"share does not exist"}`), nil
	})}

	_, _, _, err := p.ShareInfo("abc123", "")

	require.EqualError(t, err, "share access failed: share does not exist")
}

func TestShareRestoreSendsFileIDs(t *testing.T) {
	p := &PikPak{DeviceId: "device-id", JwtToken: "jwt-token"}
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "api-drive.mypikpak.com", req.URL.Host)
		require.Equal(t, "/drive/v1/share/restore", req.URL.Path)
		require.Equal(t, "application/json; charset=utf-8", req.Header.Get("Content-Type"))

		var body struct {
			Kind          string   `json:"kind"`
			ShareID       string   `json:"share_id"`
			PassCodeToken string   `json:"pass_code_token"`
			FileIDs       []string `json:"file_ids"`
		}
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Equal(t, "drive#file", body.Kind)
		require.Equal(t, "abc123", body.ShareID)
		require.Equal(t, "token-1", body.PassCodeToken)
		require.Equal(t, []string{"f1", "f2"}, body.FileIDs)
		return testHTTPResponse(http.StatusOK, `{"restore_status":"RESTORE_START","restore_task_id":"task-9"}`), nil
	})}

	status, err := p.ShareRestore("abc123", "token-1", []string{"f1", "f2"})

	require.NoError(t, err)
	require.Equal(t, "RESTORE_START", status)
}

func TestShareRestoreRejectsEmptyFileIDs(t *testing.T) {
	p := &PikPak{}
	p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Fatal("no request should be sent for empty file ids")
		return nil, nil
	})}

	_, err := p.ShareRestore("abc123", "token-1", nil)

	require.EqualError(t, err, "share contains no files to save")
}

func TestShareRestoreStructuredError(t *testing.T) {
	p := &PikPak{}
	p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return testHTTPResponse(http.StatusOK, `{"error_code":3,"error":"invalid_argument","error_description":"Request parameter error"}`), nil
	})}

	_, err := p.ShareRestore("abc123", "token-1", []string{"f1"})

	require.EqualError(t, err, "restore failed: Request parameter error")
}

func TestShareRestoreHTTPError(t *testing.T) {
	p := &PikPak{}
	p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return testHTTPResponse(http.StatusInternalServerError, `{"error_description":"boom"}`), nil
	})}

	_, err := p.ShareRestore("abc123", "token-1", []string{"f1"})

	require.ErrorContains(t, err, "HTTP 500 Internal Server Error: boom")
}

func TestShareRestoreRejectsUnknownStatus(t *testing.T) {
	p := &PikPak{}
	p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return testHTTPResponse(http.StatusOK, `{"restore_status":"RESTORE_UNKNOWN","share_status_text":"restore denied"}`), nil
	})}

	_, err := p.ShareRestore("abc123", "token-1", []string{"f1"})

	require.EqualError(t, err, "restore failed: restore denied")
}

func TestShareRestoreRejectsEmptyRestoreStatus(t *testing.T) {
	p := &PikPak{}
	p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return testHTTPResponse(http.StatusOK, `{}`), nil
	})}

	_, err := p.ShareRestore("abc123", "token-1", []string{"f1"})

	require.EqualError(t, err, "restore failed: unknown restore status")
}
