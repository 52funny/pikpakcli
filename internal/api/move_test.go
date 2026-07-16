package api

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testHTTPResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     make(http.Header),
	}
}

func TestMoveRejectsEmptyIDs(t *testing.T) {
	p := &PikPak{}
	if err := p.Move(nil, "destination"); err == nil {
		t.Fatal("Move(nil, ...) returned nil error")
	}
}

func TestMoveSendsBatchRequest(t *testing.T) {
	p := &PikPak{
		CaptchaToken: "captcha-token",
		DeviceId:     "device-id",
		JwtToken:     "jwt-token",
	}
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, http.MethodPost, req.Method)
		require.Equal(t, "api-drive.mypikpak.com", req.URL.Host)
		require.Equal(t, "/drive/v1/files:batchMove", req.URL.Path)
		require.Equal(t, "application/json", req.Header.Get("Content-Type"))
		require.Equal(t, "captcha-token", req.Header.Get("X-Captcha-Token"))
		require.Equal(t, "device-id", req.Header.Get("X-Device-Id"))
		require.Equal(t, "Bearer jwt-token", req.Header.Get("Authorization"))

		var body struct {
			IDs []string `json:"ids"`
			To  struct {
				ParentID string `json:"parent_id"`
			} `json:"to"`
		}
		require.NoError(t, json.NewDecoder(req.Body).Decode(&body))
		require.Equal(t, []string{"one", "two"}, body.IDs)
		require.Equal(t, "destination", body.To.ParentID)
		return testHTTPResponse(http.StatusOK, `{}`), nil
	})}

	require.NoError(t, p.Move([]string{"one", "two"}, "destination"))
}

func TestMoveRejectsInvalidAndHTTPErrorResponses(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		wantError  string
	}{
		{
			name:       "gateway HTML",
			statusCode: http.StatusBadGateway,
			body:       `<html>bad gateway</html>`,
			wantError:  "HTTP 502 Bad Gateway returned a non-JSON response",
		},
		{
			name:       "empty success response",
			statusCode: http.StatusOK,
			body:       "",
			wantError:  "server returned an invalid JSON response",
		},
		{
			name:       "non-object JSON",
			statusCode: http.StatusOK,
			body:       `[]`,
			wantError:  "server returned an invalid JSON response",
		},
		{
			name:       "HTTP JSON error",
			statusCode: http.StatusBadRequest,
			body:       `{"error_code":400,"error":"bad request"}`,
			wantError:  "HTTP 400 Bad Request: bad request",
		},
		{
			name:       "API error on success status",
			statusCode: http.StatusOK,
			body:       `{"error_code":123,"error":"move denied"}`,
			wantError:  "move files failed: move denied",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &PikPak{}
			p.client = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return testHTTPResponse(tt.statusCode, tt.body), nil
			})}

			err := p.Move([]string{"one"}, "destination")

			require.ErrorContains(t, err, tt.wantError)
		})
	}
}

func TestMoveRefreshesCaptchaOnceThenSucceeds(t *testing.T) {
	var moveRequests atomic.Int32
	var captchaRequests atomic.Int32
	var moveCaptchaHeaders []string

	p := &PikPak{CaptchaToken: "old-token"}
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "api-drive.mypikpak.com":
			requestNumber := moveRequests.Add(1)
			moveCaptchaHeaders = append(moveCaptchaHeaders, req.Header.Get("X-Captcha-Token"))
			if requestNumber == 1 {
				return testHTTPResponse(http.StatusOK, `{"error_code":9,"error":"captcha required"}`), nil
			}
			return testHTTPResponse(http.StatusOK, `{}`), nil
		case "user.mypikpak.com":
			captchaRequests.Add(1)
			return testHTTPResponse(http.StatusOK, `{"captcha_token":"new-token"}`), nil
		default:
			t.Fatalf("unexpected request host: %s", req.URL.Host)
			return nil, nil
		}
	})}

	err := p.Move([]string{"one"}, "destination")

	require.NoError(t, err)
	require.EqualValues(t, 2, moveRequests.Load())
	require.EqualValues(t, 1, captchaRequests.Load())
	require.Equal(t, []string{"old-token", "new-token"}, moveCaptchaHeaders)
}

func TestMoveStopsAfterOneCaptchaRefresh(t *testing.T) {
	var moveRequests atomic.Int32
	var captchaRequests atomic.Int32

	p := &PikPak{CaptchaToken: "old-token"}
	p.client = &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "api-drive.mypikpak.com":
			moveRequests.Add(1)
			return testHTTPResponse(http.StatusOK, `{"error_code":9,"error":"captcha required"}`), nil
		case "user.mypikpak.com":
			captchaRequests.Add(1)
			return testHTTPResponse(http.StatusOK, `{"captcha_token":"new-token"}`), nil
		default:
			t.Fatalf("unexpected request host: %s", req.URL.Host)
			return nil, nil
		}
	})}

	err := p.Move([]string{"one"}, "destination")

	require.EqualError(t, err, "move files failed: captcha challenge persisted after 1 refresh")
	require.EqualValues(t, 2, moveRequests.Load())
	require.EqualValues(t, 1, captchaRequests.Load())
}
