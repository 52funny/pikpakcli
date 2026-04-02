package api

import (
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

// Remaining returns the unused quota amount.
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

type TransferMessage struct {
	Transfer TransferQuotaCollection `json:"transfer"`
	Base     TransferQuotaBase       `json:"base"`
}

type TransferQuotaCollection struct {
	Offline  TransferQuota `json:"offline"`
	Download TransferQuota `json:"download"`
	Upload   TransferQuota `json:"upload"`
}

type TransferQuotaBase struct {
	Offline  TransferQuota `json:"offline"`
	Download TransferQuota `json:"download"`
	Upload   TransferQuota `json:"upload"`
}

type TransferQuota struct {
	Info        string `json:"info"`
	TotalAssets int64  `json:"total_assets"`
	Assets      int64  `json:"assets"`
	Size        int64  `json:"size"`
}

func (q TransferQuota) Remaining() int64 {
	return q.TotalAssets - q.Assets
}

// GetQuota get cloud quota
func (p *PikPak) GetQuota() (QuotaMessage, error) {
	req, err := p.newRequest("GET", "https://api-drive.mypikpak.com/drive/v1/about", nil)
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

// GetTransferQuota gets monthly transfer quota.
func (p *PikPak) GetTransferQuota() (TransferMessage, error) {
	req, err := p.newRequest("GET", "https://api-drive.mypikpak.com/vip/v1/quantity/list?type=transfer&limit=200", nil)
	if err != nil {
		return TransferMessage{}, err
	}
	bs, err := p.sendRequest(req)
	if err != nil {
		return TransferMessage{}, err
	}
	var transferMessage TransferMessage
	err = jsoniter.Unmarshal(bs, &transferMessage)
	if err != nil {
		return TransferMessage{}, err
	}
	return transferMessage, nil
}
