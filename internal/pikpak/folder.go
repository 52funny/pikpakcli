package pikpak

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"

	"github.com/52funny/pikpakcli/internal/utils"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
)

// 获取文件夹 id
// dir 可以包括 /.
// 若以 / 开头，函数会去除 /， 且会从 parent 目录开始查找
func (p *PikPak) GetDeepFolderId(parentId string, dirPath string) (string, error) {
	dirPath = utils.Slash(dirPath)
	if dirPath == "" {
		return parentId, nil
	}

	dirS := utils.SplitSeparator(dirPath)

	for _, dir := range dirS {
		id, err := p.GetFolderId(parentId, dir)
		if err != nil {
			return "", err
		}
		parentId = id
	}
	return parentId, nil
}

func (p *PikPak) GetPathFolderId(dirPath string) (string, error) {
	return p.GetDeepFolderId("", dirPath)
}

// 获取文件夹 id
// dir 不能包括 /
func (p *PikPak) GetFolderId(parentId string, dir string) (string, error) {
	// slash the dir path
	dir = utils.Slash(dir)

	value := url.Values{}
	value.Add("parent_id", parentId)
	value.Add("page_token", "")
	value.Add("with_audit", "false")
	value.Add("thumbnail_size", "SIZE_LARGE")
	value.Add("limit", "500")
	for {
		req, err := http.NewRequest("GET", fmt.Sprintf("https://api-drive.mypikpak.com/drive/v1/files?"+value.Encode()), nil)
		if err != nil {
			return "", err
		}
		req.Header.Set("Country", "CN")
		req.Header.Set("X-Peer-Id", p.DeviceId)
		req.Header.Set("X-User-Region", "1")
		req.Header.Set("X-Alt-Capability", "3")
		req.Header.Set("X-Client-Version-Code", "10083")
		req.Header.Set("X-Captcha-Token", p.CaptchaToken)
		bs, err := p.sendRequest(req)
		if err != nil {
			return "", err
		}
		files := gjson.GetBytes(bs, "files").Array()

		for _, file := range files {
			kind := file.Get("kind").String()
			name := file.Get("name").String()
			trashed := file.Get("trashed").Bool()
			if kind == "drive#folder" && name == dir && !trashed {
				return file.Get("id").String(), nil
			}
		}
		nextToken := gjson.GetBytes(bs, "next_page_token").String()
		if nextToken == "" {
			break
		}
		value.Set("page_token", nextToken)
	}
	return "", ErrNotFoundFolder
}

func (p *PikPak) GetDeepFolderOrCreateId(parentId string, dirPath string) (string, error) {
	dirPath = utils.Slash(dirPath)
	if dirPath == "" || dirPath == "." {
		return parentId, nil
	}

	dirS := utils.SplitSeparator(dirPath)

	for _, dir := range dirS {
		id, err := p.GetFolderId(parentId, dir)
		if err != nil {
			logrus.Warn("dir ", err)
			if err == ErrNotFoundFolder {
				createId, err := p.CreateFolder(parentId, dir)
				if err != nil {
					return "", err
				} else {
					logrus.Info("create dir: ", dir)
					parentId = createId
				}
			} else {
				return "", err
			}
		} else {
			parentId = id
		}
	}
	return parentId, nil
}

// Create new folder in parent folder
// parentId is parent folder id
func (p *PikPak) CreateFolder(parentId, dir string) (string, error) {
	m := map[string]interface{}{
		"kind":      "drive#folder",
		"parent_id": parentId,
		"name":      dir,
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
