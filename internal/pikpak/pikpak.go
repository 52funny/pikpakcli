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
	if conf.UseProxy {
		url, err := url.Parse(conf.Config.Proxy)
		if err != nil {
			logrus.Errorln("url parse proxy error", err)
		}
		p := http.ProxyURL(url)
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

func (p *PikPak) Login() error {
	m := make(map[string]string)
	m["client_id"] = clientID
	m["client_secret"] = clientSecret
	m["grant_type"] = "password"
	m["username"] = p.Account
	m["password"] = p.Password
	bs, err := jsoniter.Marshal(&m)
	if err != nil {
		return err
	}
	req, err := http.NewRequest("POST", "https://user.mypikpak.com/v1/auth/token", bytes.NewBuffer(bs))
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
