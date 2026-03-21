package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
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
	// initialized tracks if encryption has been initialized
	initialized = false
)

var (
	ErrEncryptionNotInitialized = errors.New("encryption not initialized")
	ErrEncryptionFailed         = errors.New("encryption failed")
	ErrDecryptionFailed         = errors.New("decryption failed")
	ErrEmptyData                = errors.New("data cannot be empty")
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
	initialized = true

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

// IsInitialized returns whether encryption has been initialized
func IsInitialized() bool {
	return initialized
}

// EncryptBytes encrypts byte data and returns encrypted bytes with version prefix
func EncryptBytes(plaintext []byte) ([]byte, error) {
	if !initialized {
		return nil, ErrEncryptionNotInitialized
	}
	if len(plaintext) == 0 {
		return nil, ErrEmptyData
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	// Prepend key version (1 byte) to the ciphertext
	result := append([]byte{byte(keyVersion)}, ciphertext...)
	return result, nil
}

// DecryptBytes decrypts encrypted bytes and returns plaintext
func DecryptBytes(ciphertext []byte) ([]byte, error) {
	if !initialized {
		return nil, ErrEncryptionNotInitialized
	}
	if len(ciphertext) == 0 {
		return nil, ErrEmptyData
	}

	if len(ciphertext) < 1 {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrDecryptionFailed)
	}

	// Extract key version
	version := int(ciphertext[0])
	if version != KeyVersion {
		return nil, fmt.Errorf("unsupported key version: %d", version)
	}

	ciphertext = ciphertext[1:]

	if len(ciphertext) < NonceSize {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrDecryptionFailed)
	}

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	nonce := ciphertext[:NonceSize]
	encrypted := ciphertext[NonceSize:]

	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

// GenerateEncryptionKey generates a new random encryption key
func GenerateEncryptionKey() (string, error) {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return "", fmt.Errorf("failed to generate key: %w", err)
	}
	return base64.URLEncoding.EncodeToString(key), nil
}

// ValidateKey validates that a key is properly formatted
func ValidateKey(key string) error {
	if key == "" {
		return ErrEncryptionNotInitialized
	}

	// Try to decode as base64
	decoded, err := base64.URLEncoding.DecodeString(key)
	if err != nil {
		// If not base64, just check it's not empty (will be hashed)
		return nil
	}

	// Check if decoded key is appropriate size
	if len(decoded) < 16 {
		return errors.New("key too short")
	}

	return nil
}
