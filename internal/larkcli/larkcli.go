package larkcli

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

type Credentials struct {
	AppID            string
	AppSecret        string
	Domain           string
	UserAccessToken  string
	RefreshToken     string
	TokenExpiresAt   int64
	RefreshExpiresAt int64
}

func (c *Credentials) TokenValid() bool {
	return c.UserAccessToken != "" && time.Now().Unix() < c.TokenExpiresAt
}

func (c *Credentials) RefreshValid() bool {
	return c.RefreshToken != "" && time.Now().Unix() < c.RefreshExpiresAt
}

type multiAppConfig struct {
	CurrentApp string      `json:"currentApp"`
	Apps       []appConfig `json:"apps"`
}

type appConfig struct {
	Name      string      `json:"name"`
	AppID     string      `json:"appId"`
	AppSecret secretInput `json:"appSecret"`
	Brand     string      `json:"brand"`
	Users     []appUser   `json:"users"`
}

type secretInput struct {
	Source string `json:"source"`
	ID     string `json:"id"`
	raw    string
}

func (s *secretInput) UnmarshalJSON(data []byte) error {
	var str string
	if json.Unmarshal(data, &str) == nil {
		s.raw = str
		return nil
	}
	type alias secretInput
	var obj alias
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	*s = secretInput(obj)
	return nil
}

type appUser struct {
	UserOpenID string `json:"userOpenId"`
	UserName   string `json:"userName"`
}

type storedUAToken struct {
	AccessToken      string `json:"accessToken"`
	RefreshToken     string `json:"refreshToken"`
	ExpiresAt        int64  `json:"expiresAt"`
	RefreshExpiresAt int64  `json:"refreshExpiresAt"`
}

func Load() (*Credentials, error) {
	configDir := os.Getenv("LARKSUITE_CLI_CONFIG_DIR")
	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configDir = filepath.Join(home, ".lark-cli")
	}

	data, err := os.ReadFile(filepath.Join(configDir, "config.json"))
	if err != nil {
		return nil, fmt.Errorf("未找到 lark-cli 配置: %w", err)
	}

	var mac multiAppConfig
	if err := json.Unmarshal(data, &mac); err != nil {
		return nil, fmt.Errorf("解析 lark-cli config.json 失败: %w", err)
	}

	if len(mac.Apps) == 0 {
		return nil, fmt.Errorf("lark-cli 未配置任何应用")
	}

	app := mac.Apps[0]
	for _, a := range mac.Apps {
		if a.AppID == mac.CurrentApp || a.Name == mac.CurrentApp {
			app = a
			break
		}
	}

	if app.AppID == "" {
		return nil, fmt.Errorf("lark-cli 应用配置缺少 appId")
	}

	masterKey, err := readMasterKey()
	if err != nil {
		return nil, fmt.Errorf("读取 lark-cli master key 失败: %w", err)
	}

	creds := &Credentials{
		AppID: app.AppID,
	}

	if app.Brand == "lark" || app.Brand == "larksuite" {
		creds.Domain = "larksuite.com"
	}

	if app.AppSecret.raw != "" {
		creds.AppSecret = app.AppSecret.raw
	} else if app.AppSecret.Source == "keychain" {
		secret, err := decryptKeychainItem(masterKey, app.AppSecret.ID)
		if err == nil {
			creds.AppSecret = string(secret)
		}
	}

	if len(app.Users) > 0 {
		tokenKey := app.AppID + ":" + app.Users[0].UserOpenID
		tokenJSON, err := decryptKeychainItem(masterKey, tokenKey)
		if err == nil {
			var token storedUAToken
			if json.Unmarshal(tokenJSON, &token) == nil {
				creds.UserAccessToken = token.AccessToken
				creds.RefreshToken = token.RefreshToken
				creds.TokenExpiresAt = token.ExpiresAt / 1000
				creds.RefreshExpiresAt = token.RefreshExpiresAt / 1000
			}
		}
	}

	return creds, nil
}

func dataDir() string {
	if env := os.Getenv("LARKSUITE_CLI_DATA_DIR"); env != "" {
		return filepath.Join(env, "lark-cli")
	}
	home, _ := os.UserHomeDir()
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "lark-cli")
	default:
		if xdg := os.Getenv("XDG_DATA_HOME"); xdg != "" {
			return filepath.Join(xdg, "lark-cli")
		}
		return filepath.Join(home, ".local", "share", "lark-cli")
	}
}

func readMasterKey() ([]byte, error) {
	dir := dataDir()
	for _, name := range []string{"master.key", "master.key.file"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err == nil && len(data) == 32 {
			return data, nil
		}
	}
	return nil, fmt.Errorf("master.key 不存在或长度不为 32 字节 (%s)", dir)
}

var safeKeyRe = regexp.MustCompile(`[^a-zA-Z0-9._-]`)

func keychainFileName(account string) string {
	return safeKeyRe.ReplaceAllString(account, "_") + ".enc"
}

func decryptKeychainItem(masterKey []byte, account string) ([]byte, error) {
	filePath := filepath.Join(dataDir(), keychainFileName(account))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	if len(data) < 12+16 {
		return nil, fmt.Errorf("加密文件过短")
	}

	iv := data[:12]
	ciphertext := data[12:]

	block, err := aes.NewCipher(masterKey)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	plaintext, err := gcm.Open(nil, iv, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("解密失败: %w", err)
	}
	return plaintext, nil
}

func Available() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	configPath := filepath.Join(home, ".lark-cli", "config.json")
	if env := os.Getenv("LARKSUITE_CLI_CONFIG_DIR"); env != "" {
		configPath = filepath.Join(env, "config.json")
	}
	_, err = os.Stat(configPath)
	return err == nil
}

func Source() string {
	return strings.Join([]string{
		"lark-cli",
		"(config: ~/.lark-cli/config.json",
		"token: " + dataDir() + "/)",
	}, " ")
}
