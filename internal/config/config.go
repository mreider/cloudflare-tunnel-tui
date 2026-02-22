package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
	"os"

	"golang.org/x/crypto/argon2"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Domain           string   `yaml:"domain"`
	CloudflaredBin   string   `yaml:"cloudflared_bin"`
	Devices          []Device `yaml:"devices"`
}

type Device struct {
	Name     string    `yaml:"name"`
	Services []Service `yaml:"services"`
}

type Service struct {
	Name           string `yaml:"name"`
	Hostname       string `yaml:"hostname"`
	Type           string `yaml:"type"` // ssh, vnc, rdp, novnc, http
	User           string `yaml:"user,omitempty"`
	Password       string `yaml:"password,omitempty"`
	ProxyLocalPort int    `yaml:"proxy_local_port,omitempty"`
	URL            string `yaml:"url,omitempty"`
}

// Argon2id parameters for key derivation
const (
	argonTime    = 3
	argonMemory  = 64 * 1024 // 64 MB
	argonThreads = 4
	argonKeyLen  = 32 // AES-256
	saltLen      = 16
)

func LoadEncrypted(path string, password string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading bundle: %w", err)
	}

	if len(data) < saltLen+12+16 { // salt + nonce + minimum ciphertext with tag
		return nil, fmt.Errorf("bundle file is too small or corrupted")
	}

	salt := data[:saltLen]
	rest := data[saltLen:]

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(rest) < nonceSize {
		return nil, fmt.Errorf("bundle file is corrupted")
	}

	nonce, ciphertext := rest[:nonceSize], rest[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decryption failed (wrong password?): %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(plaintext, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

func SaveEncrypted(cfg *Config, path string, password string) error {
	plaintext, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("serializing config: %w", err)
	}

	salt := make([]byte, saltLen)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return fmt.Errorf("generating salt: %w", err)
	}

	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)

	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("creating GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Final format: salt + nonce + ciphertext (nonce is prepended by Seal)
	out := append(salt, ciphertext...)

	if err := os.WriteFile(path, out, 0600); err != nil {
		return fmt.Errorf("writing bundle: %w", err)
	}

	return nil
}

func LoadYAML(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
