package pikpak

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/52funny/pikpakcli/conf"
	jsoniter "github.com/json-iterator/go"
	"github.com/sirupsen/logrus"
)

const userAgent = `ANDROID-com.pikcloud.pikpak/1.21.0`
const clientID = `YNxT9w7GMdWvEOKa`
const clientSecret = `dbw2OtmVEeuUvIptb1Coyg`

type PikPak struct {
	Account       string `json:"account"`
	Password      string `json:"password"`
	JwtToken      string `json:"token"`
	refreshToken  string
	CaptchaToken  string `json:"captchaToken"`
	Sub           string `json:"userId"`
	DeviceId      string `json:"deviceId"`
	RefreshSecond int64  `json:"refreshSecond"`
	client        *http.Client
}

func NewPikPak(account, password string) PikPak {
	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyFromEnvironment,
		},
	}
	if conf.Config.UseProxy() {
		proxyUrl, err := url.Parse(conf.Config.Proxy)
		if err != nil {
			logrus.Errorln("url parse proxy error", err)
		}
		p := http.ProxyURL(proxyUrl)
		client.Transport = &http.Transport{
			Proxy: p,
		}
		http.DefaultClient.Transport = &http.Transport{
			Proxy: p,
		}
	}
	n := md5.Sum([]byte(account))
	return PikPak{
		Account:  account,
		Password: password,
		DeviceId: hex.EncodeToString(n[:]),
		client:   client,
	}
}

// login 执行完整登录流程
func (p *PikPak) login() error {
	captchaToken, err := p.getCaptchaToken()
	if err != nil {
		return err
	}
	m := make(map[string]string)
	m["client_id"] = clientID
	m["client_secret"] = clientSecret
	m["grant_type"] = "password"
	m["username"] = p.Account
	m["password"] = p.Password
	m["captcha_token"] = captchaToken
	bs, err := jsoniter.Marshal(&m)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", "https://user.mypikpak.com/v1/auth/signin", bytes.NewBuffer(bs))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	bs, err = p.sendRequest(req)
	if err != nil {
		return err
	}

	error_code := jsoniter.Get(bs, "error_code").ToInt()

	if error_code != 0 {
		return fmt.Errorf("login error: %v", jsoniter.Get(bs, "error").ToString())
	}

	p.JwtToken = jsoniter.Get(bs, "access_token").ToString()
	p.refreshToken = jsoniter.Get(bs, "refresh_token").ToString()
	p.Sub = jsoniter.Get(bs, "sub").ToString()
	p.RefreshSecond = jsoniter.Get(bs, "expires_in").ToInt64()
	return nil
}

func (p *PikPak) getCaptchaToken() (string, error) {
	m := make(map[string]any)
	m["client_id"] = clientID
	m["device_id"] = p.DeviceId
	m["action"] = "POST:https://user.mypikpak.com/v1/auth/signin"
	m["meta"] = map[string]string{
		"username": p.Account,
	}
	body, err := jsoniter.Marshal(&m)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", "https://user.mypikpak.com/v1/shield/captcha/init", bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Add("Content-Type", "application/json")
	bs, err := p.sendRequest(req)
	if err != nil {
		return "", err
	}
	error_code := jsoniter.Get(bs, "error_code").ToInt()
	if error_code != 0 {
		return "", fmt.Errorf("get captcha error: %v", jsoniter.Get(bs, "error").ToString())
	}
	return jsoniter.Get(bs, "captcha_token").ToString(), nil
}

func (p *PikPak) sendRequest(req *http.Request) ([]byte, error) {
	p.setHeader(req)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	bs, err := io.ReadAll(resp.Body)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	return bs, nil
}

func (p *PikPak) setHeader(req *http.Request) {
	if p.JwtToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.JwtToken)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("X-Device-Id", p.DeviceId)
}

// Login 优先复用本地 session，必要时才走完整登录
func (p *PikPak) Login() error {
	if err := p.loadSession(); err == nil {
		if !p.isTokenExpired() {
			logrus.Debugln("session valid, skip login")
			return nil
		}
		logrus.Debugln("access_token expired, trying refresh_token")
		if err = p.RefreshToken(); err == nil {
			return p.saveSession()
		}
		logrus.Debugln("refresh failed, fallback to full login")
	}
	if err := p.login(); err != nil {
		return err
	}
	// 执行了完整登录流程，保存session
	return p.saveSession()
}
