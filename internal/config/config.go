package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/wertycn/feishu-doc/internal/crypto"
)

type Config struct {
	AppID            string `json:"app_id"`
	AppSecret        string `json:"app_secret"`
	Domain           string `json:"domain,omitempty"`
	UserAccessToken  string `json:"user_access_token,omitempty"`
	RefreshToken     string `json:"refresh_token,omitempty"`
	TokenExpiresAt   int64  `json:"token_expires_at,omitempty"`
	RefreshExpiresAt int64  `json:"refresh_expires_at,omitempty"`
}

func (c *Config) TokenValid() bool {
	return c.UserAccessToken != "" && time.Now().Unix() < c.TokenExpiresAt
}

func (c *Config) RefreshValid() bool {
	return c.RefreshToken != "" && time.Now().Unix() < c.RefreshExpiresAt
}

func Dir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".feishu-doc")
}

func path() string {
	return filepath.Join(Dir(), "config.enc")
}

func Save(cfg *Config) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	encrypted, err := crypto.Encrypt(data)
	if err != nil {
		return fmt.Errorf("加密失败: %w", err)
	}
	if err := os.MkdirAll(Dir(), 0700); err != nil {
		return err
	}
	return os.WriteFile(path(), encrypted, 0600)
}

func Load() (*Config, error) {
	data, err := os.ReadFile(path())
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("未找到配置，请先运行: feishu-doc config set --app-id <ID> --app-secret <SECRET>")
		}
		return nil, err
	}
	decrypted, err := crypto.Decrypt(data)
	if err != nil {
		return nil, fmt.Errorf("解密失败(配置文件可能已损坏): %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(decrypted, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func Mask(s string) string {
	if len(s) <= 6 {
		return "***"
	}
	return s[:3] + "****" + s[len(s)-3:]
}
