package pikpak

import (
	"net/http"
	"strconv"

	jsoniter "github.com/json-iterator/go"
)

type QuotaMessage struct {
	Kind      string `json:"kind"`
	Quota     Quota  `json:"quota"`
	ExpiresAt string `json:"expires_at"`
	Quotas    Quotas `json:"quotas"`
}
type Quota struct {
	Kind           string `json:"kind"`
	Limit          string `json:"limit"`
	Usage          string `json:"usage"`
	UsageInTrash   string `json:"usage_in_trash"`
	PlayTimesLimit string `json:"play_times_limit"`
	PlayTimesUsage string `json:"play_times_usage"`
}

// Remaining 剩余额度
func (q Quota) Remaining() (int64, error) {
	limit, err := strconv.ParseInt(q.Limit, 10, 64)
	if err != nil {
		return 0, err
	}
	usage, err := strconv.ParseInt(q.Usage, 10, 64)
	if err != nil {
		return 0, err
	}
	return limit - usage, nil
}

type Quotas struct {
	CloudDownload Quota `json:"cloud_download"`
}

// GetQuota get cloud quota
func (p *PikPak) GetQuota() (QuotaMessage, error) {
	req, err := http.NewRequest("GET", "https://api-drive.mypikpak.com/drive/v1/about", nil)
	if err != nil {
		return QuotaMessage{}, err
	}
	bs, err := p.sendRequest(req)
	if err != nil {
		return QuotaMessage{}, err
	}
	var quotaMessage QuotaMessage
	err = jsoniter.Unmarshal(bs, &quotaMessage)
	if err != nil {
		return QuotaMessage{}, err
	}
	return quotaMessage, nil
}
