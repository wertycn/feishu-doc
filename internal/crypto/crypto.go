package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
)

func saltDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".feishu-doc")
}

func loadOrCreateSalt() []byte {
	sp := filepath.Join(saltDir(), ".salt")
	data, err := os.ReadFile(sp)
	if err == nil && len(data) == 32 {
		return data
	}
	salt := make([]byte, 32)
	io.ReadFull(rand.Reader, salt)
	os.MkdirAll(saltDir(), 0700)
	os.WriteFile(sp, salt, 0600)
	return salt
}

func deriveKey() []byte {
	salt := loadOrCreateSalt()
	hostname, _ := os.Hostname()
	u, _ := user.Current()
	username := ""
	if u != nil {
		username = u.Username
	}
	material := fmt.Sprintf("feishu-doc:v2:%s:%s:%s", hostname, username, hex.EncodeToString(salt))
	hash := sha256.Sum256([]byte(material))
	return hash[:]
}

func Encrypt(plaintext []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveKey())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func Decrypt(ciphertext []byte) ([]byte, error) {
	block, err := aes.NewCipher(deriveKey())
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("密文数据损坏")
	}
	return gcm.Open(nil, ciphertext[:nonceSize], ciphertext[nonceSize:], nil)
}
