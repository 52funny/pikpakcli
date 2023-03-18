package pikpak

import (
	"net/http"

	jsoniter "github.com/json-iterator/go"
)

type quotaMessage struct {
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

type Quotas struct {
}

// get cloud quota
func (p *PikPak) GetQuota() (Quota, error) {
	req, err := http.NewRequest("GET", "https://api-drive.mypikpak.com/drive/v1/about", nil)
	if err != nil {
		return Quota{}, err
	}
	bs, err := p.sendRequest(req)
	if err != nil {
		return Quota{}, err
	}
	var quotaMessage quotaMessage
	err = jsoniter.Unmarshal(bs, &quotaMessage)
	if err != nil {
		return Quota{}, err
	}
	return quotaMessage.Quota, nil
}
