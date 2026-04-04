package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
	"time"
	"unsafe"

	"github.com/52funny/pikpakcli/internal/logx"
	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/tidwall/gjson"
)

type FileStat struct {
	Kind          string    `json:"kind"`
	ID            string    `json:"id"`
	ParentID      string    `json:"parent_id"`
	Name          string    `json:"name"`
	UserID        string    `json:"user_id"`
	Size          string    `json:"size"`
	FileExtension string    `json:"file_extension"`
	MimeType      string    `json:"mime_type"`
	CreatedTime   time.Time `json:"created_time"`
	ModifiedTime  time.Time `json:"modified_time"`
	IconLink      string    `json:"icon_link"`
	ThumbnailLink string    `json:"thumbnail_link"`
	Md5Checksum   string    `json:"md5_checksum"`
	Hash          string    `json:"hash"`
	Phase         string    `json:"phase"`
}
type File struct {
	FileStat
	Revision       string `json:"revision"`
	Starred        bool   `json:"starred"`
	WebContentLink string `json:"web_content_link"`
	Links          struct {
		ApplicationOctetStream struct {
			URL    string    `json:"url"`
			Token  string    `json:"token"`
			Expire time.Time `json:"expire"`
		} `json:"application/octet-stream"`
	} `json:"links"`
	Audit struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Title   string `json:"title"`
	} `json:"audit"`
	Medias []struct {
		MediaID   string      `json:"media_id"`
		MediaName string      `json:"media_name"`
		Video     interface{} `json:"video"`
		Link      struct {
			URL    string    `json:"url"`
			Token  string    `json:"token"`
			Expire time.Time `json:"expire"`
		} `json:"link"`
		NeedMoreQuota  bool          `json:"need_more_quota"`
		VipTypes       []interface{} `json:"vip_types"`
		RedirectLink   string        `json:"redirect_link"`
		IconLink       string        `json:"icon_link"`
		IsDefault      bool          `json:"is_default"`
		Priority       int           `json:"priority"`
		IsOrigin       bool          `json:"is_origin"`
		ResolutionName string        `json:"resolution_name"`
		IsVisible      bool          `json:"is_visible"`
		Category       string        `json:"category"`
	} `json:"medias"`
	Trashed     bool   `json:"trashed"`
	DeleteTime  string `json:"delete_time"`
	OriginalURL string `json:"original_url"`
	Params      struct {
		Platform     string `json:"platform"`
		PlatformIcon string `json:"platform_icon"`
	} `json:"params"`
	OriginalFileIndex int           `json:"original_file_index"`
	Space             string        `json:"space"`
	Apps              []interface{} `json:"apps"`
	Writable          bool          `json:"writable"`
	FolderType        string        `json:"folder_type"`
	Collection        interface{}   `json:"collection"`
	ctx               context.Context
}

type fileListResult struct {
	NextPageToken string     `json:"next_page_token"`
	Files         []FileStat `json:"files"`
}

const maxListRetries = 3

func (p *PikPak) GetFolderFileStatList(parentId string) ([]FileStat, error) {
	filters := `{"trashed":{"eq":false}}`
	query := url.Values{}
	query.Add("thumbnail_size", "SIZE_MEDIUM")
	query.Add("limit", "500")
	query.Add("parent_id", parentId)
	query.Add("with_audit", "false")
	query.Add("filters", filters)
	fileList := make([]FileStat, 0)

	for {
		bs, err := p.getFolderFileStatPage(query)
		if err != nil {
			return fileList, err
		}
		error_code := gjson.Get(*(*string)(unsafe.Pointer(&bs)), "error_code").Int()
		if error_code == 9 {
			err = p.AuthCaptchaToken("GET:/drive/v1/files")
			if err != nil {
				return fileList, err
			}
		}
		var result fileListResult
		err = json.Unmarshal(bs, &result)
		if err != nil {
			return fileList, err
		}
		fileList = append(fileList, result.Files...)
		if result.NextPageToken == "" {
			break
		}
		query.Set("page_token", result.NextPageToken)
	}
	return fileList, nil
}

func (p *PikPak) getFolderFileStatPage(query url.Values) ([]byte, error) {
	var lastErr error
	for attempt := 0; attempt < maxListRetries; attempt++ {
		req, err := p.newRequest("GET", "https://api-drive.mypikpak.com/drive/v1/files?"+query.Encode(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("X-Captcha-Token", p.CaptchaToken)
		req.Header.Set("Content-Type", "application/json")
		bs, err := p.sendRequest(req)
		if err == nil {
			return bs, nil
		}
		lastErr = err
		if !isRetryableListError(err) || attempt == maxListRetries-1 {
			break
		}
		logx.Warnf("transfer", "List folder interrupted, retrying (%d/%d): %v", attempt+1, maxListRetries-1, err)
		time.Sleep(time.Duration(attempt+1) * 200 * time.Millisecond)
	}
	return nil, lastErr
}

func isRetryableListError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	message := strings.ToLower(err.Error())
	return strings.Contains(message, "unexpected eof") ||
		strings.Contains(message, "connection reset by peer") ||
		strings.Contains(message, "connection closed") ||
		strings.Contains(message, "broken pipe")
}

// Find FileState similar to name in the parentId directory
func (p *PikPak) GetFileStat(parentId string, name string) (FileStat, error) {
	stats, err := p.GetFolderFileStatList(parentId)
	if err != nil {
		return FileStat{}, err
	}
	for _, stat := range stats {
		if stat.Name == name {
			return stat, nil
		}
	}
	return FileStat{}, errors.New("file not found")
}

func (p *PikPak) GetFileByPath(path string) (FileStat, error) {
	parentPath, name := utils.SplitRemotePath(path)
	if name == "" {
		return FileStat{}, errors.New("cannot get info of root directory")
	}

	parentID := ""
	var err error
	if parentPath != "" {
		parentID, err = p.GetPathFolderId(parentPath)
		if err != nil {
			return FileStat{}, err
		}
	}

	return p.GetFileStat(parentID, name)
}

func (p *PikPak) GetFile(fileId string) (File, error) {
	var fileInfo File
	query := url.Values{}
	query.Add("thumbnail_size", "SIZE_MEDIUM")
	req, err := p.newRequest("GET", "https://api-drive.mypikpak.com/drive/v1/files/"+fileId+"?"+query.Encode(), nil)
	if err != nil {
		return fileInfo, err
	}
	req.Header.Set("X-Captcha-Token", p.CaptchaToken)
	req.Header.Set("X-Device-Id", p.DeviceId)
	bs, err := p.sendRequest(req)
	if err != nil {
		return fileInfo, err
	}

	error_code := gjson.Get(*(*string)(unsafe.Pointer(&bs)), "error_code").Int()
	if error_code != 0 {
		if error_code == 9 {
			err = p.AuthCaptchaToken("GET:/drive/v1/files")
			if err != nil {
				return fileInfo, err
			}
		}
		err = errors.New(gjson.Get(*(*string)(unsafe.Pointer(&bs)), "error").String() + ":" + fileId)
		return fileInfo, err
	}
	err = json.Unmarshal(bs, &fileInfo)
	if err != nil {
		return fileInfo, err
	}
	fileInfo.ctx = p.requestContext()
	return fileInfo, err
}

func (p *PikPak) DeleteFile(fileId string) error {
START:
	req, err := p.newRequest("DELETE", "https://api-drive.mypikpak.com/drive/v1/files/"+fileId, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Captcha-Token", p.CaptchaToken)
	req.Header.Set("X-Device-Id", p.DeviceId)
	bs, err := p.sendRequest(req)
	if err != nil {
		return err
	}
	error_code := gjson.GetBytes(bs, "error_code").Int()
	if error_code != 0 {
		if error_code == 9 {
			err = p.AuthCaptchaToken("DELETE:/drive/v1/files")
			if err != nil {
				return err
			}
			goto START
		}
		return fmt.Errorf("%s: %s", gjson.GetBytes(bs, "error").String(), fileId)
	}
	return nil
}

func (p *PikPak) Rename(fileId string, newName string) error {
	if newName == "" {
		return errors.New("new name cannot be empty")
	}

	apiURL := "https://api-drive.mypikpak.com/drive/v1/files/" + fileId
	body := map[string]string{"name": newName}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

START:
	req, err := p.newRequest("PATCH", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Captcha-Token", p.CaptchaToken)
	req.Header.Set("X-Device-Id", p.DeviceId)
	bs, err := p.sendRequest(req)
	if err != nil {
		return err
	}

	errorCode := gjson.GetBytes(bs, "error_code").Int()
	if errorCode != 0 {
		if errorCode == 9 {
			err = p.AuthCaptchaToken("PATCH:/drive/v1/files")
			if err != nil {
				return err
			}
			goto START
		}
		return fmt.Errorf("%s: %s", gjson.GetBytes(bs, "error").String(), fileId)
	}

	return nil
}
