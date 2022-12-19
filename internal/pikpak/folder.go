package pikpak

import (
	"bytes"
	"fmt"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/tidwall/gjson"
)

func (p *PikPak) CreateFolder(parentId, path string) (string, error) {
	m := map[string]interface{}{
		"kind":      "drive#folder",
		"parent_id": parentId,
		"name":      path,
	}
	bs, err := jsoniter.Marshal(&m)
	if err != nil {
		return "", err
	}
START:
	req, err := http.NewRequest("POST", "https://api-drive.mypikpak.com/drive/v1/files", bytes.NewBuffer(bs))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.Header.Set("Product_flavor_name", "cha")
	req.Header.Set("X-Captcha-Token", p.CaptchaToken)
	req.Header.Set("X-Client-Version-Code", "10083")
	req.Header.Set("X-Peer-Id", p.DeviceId)
	req.Header.Set("X-User-Region", "1")
	req.Header.Set("X-Alt-Capability", "3")
	req.Header.Set("Country", "CN")
	bs, err = p.sendRequest(req)
	if err != nil {
		return "", err
	}
	error_code := gjson.GetBytes(bs, "error_code").Int()
	if error_code != 0 {
		if error_code == 9 {
			err := p.AuthCaptchaToken("POST:/drive/v1/files")
			if err != nil {
				return "", err
			}
			goto START
		}
		return "", fmt.Errorf("create folder error: %s", jsoniter.Get(bs, "error").ToString())
	}
	id := gjson.GetBytes(bs, "file.id").String()
	return id, nil
}
