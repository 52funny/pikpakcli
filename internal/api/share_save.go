package api

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"github.com/52funny/pikpakcli/internal/logx"
	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
)

// ShareInfo 获取分享信息与 pass_code_token（需提取码时传 passCode）
// 返回：pass_code_token、分享标题、顶层文件ID列表
//
// 分页采用两阶段协议：
//  1. GET /drive/v1/share 校验分享并取得 pass_code_token 与元数据
//  2. GET /drive/v1/share/detail 通过 page_token 分页拉取顶层文件
func (p *PikPak) ShareInfo(shareId, passCode string) (passCodeToken, title string, fileIDs []string, err error) {
	// 阶段一：访问分享，取得 pass_code_token 与元数据
	value := url.Values{}
	value.Add("limit", "100")
	value.Add("share_id", shareId)
	if passCode != "" {
		value.Add("pass_code", passCode)
	}
	bs, statusCode, err := p.shareGet("/drive/v1/share", value)
	if err != nil {
		return "", "", nil, err
	}
	if err := shareStatusError(statusCode, bs, "share access failed"); err != nil {
		return "", "", nil, err
	}
	passCodeToken = gjson.GetBytes(bs, "pass_code_token").String()
	title = gjson.GetBytes(bs, "title").String()

	// 阶段二：通过 /drive/v1/share/detail 分页拉取全部顶层文件 ID
	fileIDs = make([]string, 0)
	pageToken := ""
	seen := map[string]bool{}
	for {
		if pageToken != "" {
			if seen[pageToken] {
				return "", "", nil, fmt.Errorf("share access failed: pagination stuck on next_page_token %q", pageToken)
			}
			seen[pageToken] = true
		}
		detail := url.Values{}
		detail.Add("limit", "100")
		detail.Add("share_id", shareId)
		detail.Add("pass_code_token", passCodeToken)
		detail.Add("page_token", pageToken)
		bs, statusCode, err := p.shareGet("/drive/v1/share/detail", detail)
		if err != nil {
			return "", "", nil, err
		}
		if err := shareStatusError(statusCode, bs, "share access failed"); err != nil {
			return "", "", nil, err
		}
		gjson.GetBytes(bs, "files.#.id").ForEach(func(_, v gjson.Result) bool {
			if id := v.String(); id != "" {
				fileIDs = append(fileIDs, id)
			}
			return true
		})
		pageToken = gjson.GetBytes(bs, "next_page_token").String()
		if pageToken == "" {
			break
		}
	}
	return passCodeToken, title, fileIDs, nil
}

// ShareRestore 转存分享到自己的网盘（默认 Pack From Shared）
// fileIDs 为分享中需要转存的顶层文件/文件夹ID，由 ShareInfo 获取
// 返回 restore 状态（RESTORE_START 表示转存任务已启动）
func (p *PikPak) ShareRestore(shareId, passCodeToken string, fileIDs []string) (string, error) {
	if len(fileIDs) == 0 {
		return "", fmt.Errorf("share contains no files to save")
	}
	m := map[string]interface{}{
		"kind":            "drive#file",
		"share_id":        shareId,
		"pass_code_token": passCodeToken,
		"file_ids":        fileIDs,
	}
	bs, err := jsoniter.Marshal(&m)
	if err != nil {
		return "", err
	}
	req, err := p.newRequest("POST", "https://api-drive.mypikpak.com/drive/v1/share/restore", bytes.NewBuffer(bs))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Country", "CN")
	req.Header.Set("X-Peer-Id", p.DeviceId)
	req.Header.Set("X-User-Region", "1")
	req.Header.Set("X-Alt-Capability", "3")
	req.Header.Set("X-Client-Version-Code", "10083")
	resp, statusCode, err := p.sendRequestWithStatus(req)
	if err != nil {
		return "", err
	}
	if err := shareAPIError(statusCode, resp); err != nil {
		return "", fmt.Errorf("restore failed: %w", err)
	}
	status := gjson.GetBytes(resp, "restore_status").String()
	if status == "" || status == "RESTORE_UNKNOWN" {
		msg := shareErrorReason(resp)
		if msg == "" {
			msg = gjson.GetBytes(resp, "share_status_text").String()
		}
		if msg == "" {
			msg = "unknown restore status"
		}
		return "", fmt.Errorf("restore failed: %s", msg)
	}
	taskID := gjson.GetBytes(resp, "restore_task_id").String()
	logx.Debug("share_restore", "status="+status+" task="+taskID)
	return status, nil
}

// shareGet performs a GET against a share API endpoint with the given query.
func (p *PikPak) shareGet(endpoint string, query url.Values) ([]byte, int, error) {
	req, err := p.newRequest("GET", "https://api-drive.mypikpak.com"+endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Country", "CN")
	req.Header.Set("X-Peer-Id", p.DeviceId)
	req.Header.Set("X-User-Region", "1")
	req.Header.Set("X-Alt-Capability", "3")
	req.Header.Set("X-Client-Version-Code", "10083")
	return p.sendRequestWithStatus(req)
}

// shareStatusError wraps shareAPIError and validates share_status for the
// share listing endpoints, so the server's real reason is never dropped.
func shareStatusError(statusCode int, bs []byte, prefix string) error {
	if err := shareAPIError(statusCode, bs); err != nil {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	if status := gjson.GetBytes(bs, "share_status").String(); status != "OK" {
		msg := gjson.GetBytes(bs, "share_status_text").String()
		if msg == "" {
			msg = status
		}
		return fmt.Errorf("%s: %s", prefix, msg)
	}
	return nil
}

// shareAPIError builds an error from a share endpoint response, preferring
// the structured error fields so the server's reason is preserved.
func shareAPIError(statusCode int, bs []byte) error {
	if statusCode < http.StatusOK || statusCode >= http.StatusMultipleChoices {
		if msg := shareErrorReason(bs); msg != "" {
			return fmt.Errorf("HTTP %d %s: %s", statusCode, http.StatusText(statusCode), msg)
		}
		return fmt.Errorf("HTTP %d %s", statusCode, http.StatusText(statusCode))
	}
	if code := gjson.GetBytes(bs, "error_code").Int(); code != 0 {
		if msg := shareErrorReason(bs); msg != "" {
			return fmt.Errorf("%s", msg)
		}
		return fmt.Errorf("error code %d", code)
	}
	return nil
}

// shareErrorReason returns the server-provided error message, preferring
// the most descriptive structured field.
func shareErrorReason(bs []byte) string {
	for _, key := range []string{"error_description", "error"} {
		if msg := gjson.GetBytes(bs, key).String(); msg != "" {
			return msg
		}
	}
	return ""
}
