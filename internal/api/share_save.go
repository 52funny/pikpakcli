package api

import (
	"bytes"
	"fmt"
	"net/url"

	"github.com/52funny/pikpakcli/internal/logx"
	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
)

// ShareInfo 获取分享信息与 pass_code_token（需提取码时传 passCode）
// 返回：pass_code_token、分享标题、顶层文件ID列表（分页拉取全部）
func (p *PikPak) ShareInfo(shareId, passCode string) (passCodeToken, title string, fileIDs []string, err error) {
	fileIDs = make([]string, 0)
	nextPageToken := ""
	for {
		value := url.Values{}
		value.Add("limit", "100")
		value.Add("share_id", shareId)
		if passCode != "" {
			value.Add("pass_code", passCode)
		}
		if nextPageToken != "" {
			value.Add("next_page_token", nextPageToken)
		}
		req, err := p.newRequest("GET", "https://api-drive.mypikpak.com/drive/v1/share?"+value.Encode(), nil)
		if err != nil {
			return "", "", nil, err
		}
		req.Header.Set("Country", "CN")
		req.Header.Set("X-Peer-Id", p.DeviceId)
		req.Header.Set("X-User-Region", "1")
		req.Header.Set("X-Alt-Capability", "3")
		req.Header.Set("X-Client-Version-Code", "10083")
		bs, err := p.sendRequest(req)
		if err != nil {
			return "", "", nil, err
		}
		status := gjson.GetBytes(bs, "share_status").String()
		if status != "OK" {
			msg := gjson.GetBytes(bs, "share_status_text").String()
			if msg == "" {
				msg = status
			}
			return "", "", nil, fmt.Errorf("share access failed: %s", msg)
		}
		passCodeToken = gjson.GetBytes(bs, "pass_code_token").String()
		title = gjson.GetBytes(bs, "title").String()
		gjson.GetBytes(bs, "files.#.id").ForEach(func(_, v gjson.Result) bool {
			if id := v.String(); id != "" {
				fileIDs = append(fileIDs, id)
			}
			return true
		})
		nextPageToken = gjson.GetBytes(bs, "next_page_token").String()
		if nextPageToken == "" {
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
	resp, err := p.sendRequest(req)
	if err != nil {
		return "", err
	}
	status := gjson.GetBytes(resp, "restore_status").String()
	if status == "" || status == "RESTORE_UNKNOWN" {
		return "", fmt.Errorf("restore failed: %s", gjson.GetBytes(resp, "share_status_text").String())
	}
	taskID := gjson.GetBytes(resp, "restore_task_id").String()
	logx.Debug("share_restore", "status="+status+" task="+taskID)
	return status, nil
}
