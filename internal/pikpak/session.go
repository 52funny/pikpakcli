package pikpak

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/52funny/pikpakcli/internal/utils"
	"github.com/sirupsen/logrus"
)

const sessionExpirySkew = 5 * 60

// sessionData 是持久化到磁盘的数据结构
type sessionData struct {
	JwtToken     string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Sub          string `json:"sub"`
	// ExpiresAt 是 access_token 的过期 Unix 时间戳（秒）
	ExpiresAt int64 `json:"expires_at"`
}

// saveSession 将当前 token 信息持久化到本地文件
func (p *PikPak) saveSession() error {
	path, err := sessionFile(p.Account)
	if err != nil {
		return err
	}
	if err := utils.CreateDirIfNotExist(filepath.Dir(path)); err != nil {
		return fmt.Errorf("create session dir error: %w", err)
	}
	data := sessionData{
		JwtToken:     p.JwtToken,
		RefreshToken: p.refreshToken,
		Sub:          p.Sub,
		// RefreshSecond 是服务端返回的 expires_in（秒），提前 5 分钟视为过期
		ExpiresAt: time.Now().Unix() + p.RefreshSecond - sessionExpirySkew,
	}

	bs, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal session error: %w", err)
	}
	if err = os.WriteFile(path, bs, 0600); err != nil {
		return fmt.Errorf("write session file error: %w", err)
	}
	logrus.Debugln("session saved to", path)
	return nil
}

// loadSession 从本地文件加载 token，并写回到 PikPak 实例
// 如果文件不存在或账号不匹配，返回 error
func (p *PikPak) loadSession() error {
	path, err := sessionFile(p.Account)
	if err != nil {
		return err
	}
	bs, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read session file error: %w", err)
	}
	var data sessionData
	if err = json.Unmarshal(bs, &data); err != nil {
		return fmt.Errorf("unmarshal session error: %w", err)
	}

	p.JwtToken = data.JwtToken
	p.refreshToken = data.RefreshToken
	p.Sub = data.Sub
	p.RefreshSecond = data.ExpiresAt - time.Now().Unix()
	logrus.Debugln("session loaded from", path)
	return nil
}

// isTokenExpired 判断 access_token 是否已过期（或即将过期）
// RefreshSecond 在 loadSession 后表示距过期的剩余秒数
func (p *PikPak) isTokenExpired() bool {
	return p.RefreshSecond <= 0
}

func (p *PikPak) saveSessionBestEffort() {
	if err := p.saveSession(); err != nil {
		logrus.Warnln("save session failed:", err)
	}
}

func sessionFile(account string) (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("get config dir error: %w", err)
	}
	hash := md5.Sum([]byte(account))
	filename := fmt.Sprintf("session_%s.json", hex.EncodeToString(hash[:]))
	return filepath.Join(configDir, "pikpakcli", filename), nil
}
