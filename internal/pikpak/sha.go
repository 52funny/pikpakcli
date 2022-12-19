package pikpak

import (
	"bytes"
	"fmt"
	"net/http"

	jsoniter "github.com/json-iterator/go"
)

func (p *PikPak) CreateShaFile(parentId, fileName, size, sha string) error {
	m := map[string]interface{}{
		"body": map[string]string{
			"duration": "",
			"width":    "",
			"height":   "",
		},
		"kind":        "drive#file",
		"name":        fileName,
		"size":        size,
		"hash":        sha,
		"upload_type": "UPLOAD_TYPE_RESUMABLE",
		"objProvider": map[string]string{
			"provider": "UPLOAD_TYPE_UNKNOWN",
		},
	}
	if parentId != "" {
		m["parent_id"] = parentId
	}
	bs, err := jsoniter.Marshal(&m)
	if err != nil {
		return err
	}
START:
	req, err := http.NewRequest("POST", "https://api-drive.mypikpak.com/drive/v1/files", bytes.NewBuffer(bs))
	if err != nil {
		return err
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
		return err
	}
	error_code := jsoniter.Get(bs, "error_code").ToInt()
	if error_code != 0 {
		if error_code == 9 {
			err := p.AuthCaptchaToken("POST:/drive/v1/files")
			if err != nil {
				return err
			}
			goto START
		}
		return fmt.Errorf("upload file error: %s", jsoniter.Get(bs, "error").ToString())
	}
	// logrus.Debug(string(bs))
	file := jsoniter.Get(bs, "file")
	phase := file.Get("phase").ToString()
	if phase == "PHASE_TYPE_COMPLETE" {
		return nil
	} else {
		return fmt.Errorf("create file error: %s", phase)
	}
}
