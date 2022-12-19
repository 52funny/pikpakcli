package pikpak

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"time"
	"unsafe"

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
}

type fileListResult struct {
	NextPageToken string     `json:"next_page_token"`
	Files         []FileStat `json:"files"`
}

func (p *PikPak) GetFolderFileStatList(parentId string) ([]FileStat, error) {
	filters := `{"trashed":{"eq":false}}`
	query := url.Values{}
	query.Add("thumbnail_size", "SIZE_MEDIUM")
	query.Add("limit", "100")
	query.Add("parent_id", parentId)
	query.Add("with_audit", "false")
	query.Add("filters", filters)
	fileList := make([]FileStat, 0)

	for {
		// query.Add("filters", filters)
		req, err := http.NewRequest("GET", "https://api-drive.mypikpak.com/drive/v1/files?"+query.Encode(), nil)
		if err != nil {
			return fileList, err
		}
		req.Header.Set("X-Captcha-Token", p.CaptchaToken)
		req.Header.Set("Content-Type", "application/json")
		bs, err := p.sendRequest(req)
		if err != nil {
			return fileList, err
		}
		error_code := gjson.Get(*(*string)(unsafe.Pointer(&bs)), "error_code").Int()
		if error_code == 9 {
			err := p.AuthCaptchaToken("GET:/drive/v1/files")
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

func (p *PikPak) GetFile(fileId string) (File, error) {
	var fileInfo File
	query := url.Values{}
	query.Add("thumbnail_size", "SIZE_MEDIUM")
	req, err := http.NewRequest("GET", "https://api-drive.mypikpak.com/drive/v1/files/"+fileId+"?"+query.Encode(), nil)
	if err != nil {
		return fileInfo, nil
	}
	req.Header.Set("X-Captcha-Token", p.CaptchaToken)
	req.Header.Set("X-Device-Id", p.DeviceId)
	bs, err := p.sendRequest(req)
	if err != nil {
		return fileInfo, nil
	}

	error_code := gjson.Get(*(*string)(unsafe.Pointer(&bs)), "error_code").Int()
	if error_code != 0 {
		if error_code == 9 {
			err := p.AuthCaptchaToken("GET:/drive/v1/files")
			if err != nil {
				return fileInfo, err
			}
		}
		err := errors.New(gjson.Get(*(*string)(unsafe.Pointer(&bs)), "error").String() + ":" + fileId)
		return fileInfo, err
	}
	err = json.Unmarshal(bs, &fileInfo)
	if err != nil {
		return fileInfo, err
	}
	return fileInfo, err
}
