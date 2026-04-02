package api

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

// sessionData is the on-disk representation of cached auth tokens.
type sessionData struct {
	JwtToken     string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	Sub          string `json:"sub"`
	// ExpiresAt stores the access token expiration time as a Unix timestamp in seconds.
	ExpiresAt int64 `json:"expires_at"`
}

// saveSession persists the current token state to the local session file.
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
		// Treat the token as expired slightly early to avoid using a near-expiry session.
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

// loadSession restores cached tokens from disk into the current client.
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

// isTokenExpired reports whether the cached access token should be treated as expired.
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
