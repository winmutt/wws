package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
)

const (
	// KeyVersion is the current encryption key version
	KeyVersion = 1
	// NonceSize is the size of the nonce for AES-GCM
	NonceSize = 12
)

var (
	// encryptionKey is the current encryption key
	encryptionKey []byte
	// keyVersion tracks which key was used for encryption
	keyVersion = KeyVersion
)

func InitEncryption() error {
	key := os.Getenv("WWS_ENCRYPTION_KEY")
	if key == "" {
		// Fallback to GitHub client secret if no encryption key is set
		key = os.Getenv("GITHUB_CLIENT_SECRET")
	}

	if key == "" {
		return fmt.Errorf("no encryption key configured")
	}

	// Derive a 32-byte key from the provided key
	hash := sha256.Sum256([]byte(key))
	encryptionKey = hash[:]
	keyVersion = KeyVersion

	return nil
}

// Encrypt encrypts plaintext using AES-GCM
func Encrypt(plaintext string) (string, error) {
	if len(encryptionKey) == 0 {
		return "", fmt.Errorf("encryption key not initialized")
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)

	// Prepend key version (1 byte) to the ciphertext
	result := append([]byte{byte(keyVersion)}, ciphertext...)

	return base64.URLEncoding.EncodeToString(result), nil
}

// Decrypt decrypts ciphertext using AES-GCM
func Decrypt(ciphertext string) (string, error) {
	if len(encryptionKey) == 0 {
		return "", fmt.Errorf("encryption key not initialized")
	}

	data, err := base64.URLEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	if len(data) < 1 {
		return "", fmt.Errorf("invalid ciphertext: too short")
	}

	// Extract key version
	version := int(data[0])
	if version != KeyVersion {
		return "", fmt.Errorf("unsupported key version: %d", version)
	}

	data = data[1:]

	if len(data) < NonceSize {
		return "", fmt.Errorf("invalid ciphertext: nonce missing")
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := data[:NonceSize]
	encrypted := data[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// GetKeyVersion returns the current key version
func GetKeyVersion() int {
	return keyVersion
}
