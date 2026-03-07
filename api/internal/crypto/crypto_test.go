package crypto

import (
	"os"
	"testing"
)

func TestInitEncryption(t *testing.T) {
	os.Setenv("WWS_ENCRYPTION_KEY", "test-key-123")
	defer os.Unsetenv("WWS_ENCRYPTION_KEY")

	err := InitEncryption()
	if err != nil {
		t.Fatalf("InitEncryption failed: %v", err)
	}

	if len(encryptionKey) == 0 {
		t.Error("encryption key not set after initialization")
	}

	if keyVersion != KeyVersion {
		t.Errorf("expected key version %d, got %d", KeyVersion, keyVersion)
	}
}

func TestInitEncryptionFallback(t *testing.T) {
	os.Setenv("GITHUB_CLIENT_SECRET", "github-secret-456")
	defer os.Unsetenv("GITHUB_CLIENT_SECRET")

	err := InitEncryption()
	if err != nil {
		t.Fatalf("InitEncryption fallback failed: %v", err)
	}

	if len(encryptionKey) == 0 {
		t.Error("encryption key not set with fallback")
	}
}

func TestInitEncryptionNoKey(t *testing.T) {
	os.Unsetenv("WWS_ENCRYPTION_KEY")
	os.Unsetenv("GITHUB_CLIENT_SECRET")

	err := InitEncryption()
	if err == nil {
		t.Error("expected error when no encryption key configured")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	os.Setenv("WWS_ENCRYPTION_KEY", "test-key-789")
	InitEncryption()

	plaintext := "secret-token-12345"

	encrypted, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if encrypted == "" {
		t.Error("encrypted result is empty")
	}

	if encrypted == plaintext {
		t.Error("encrypted result should not match plaintext")
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("decrypted '%s' does not match original '%s'", decrypted, plaintext)
	}
}

func TestEncryptMultipleTimes(t *testing.T) {
	os.Setenv("WWS_ENCRYPTION_KEY", "test-key-multiple")
	InitEncryption()

	plaintext := "token-value"

	encrypted1, _ := Encrypt(plaintext)
	encrypted2, _ := Encrypt(plaintext)

	if encrypted1 == encrypted2 {
		t.Error("encrypting same plaintext twice should produce different ciphertexts")
	}

	decrypted1, _ := Decrypt(encrypted1)
	decrypted2, _ := Decrypt(encrypted2)

	if decrypted1 != plaintext || decrypted2 != plaintext {
		t.Error("both decrypted values should match original")
	}
}

func TestDecryptInvalidBase64(t *testing.T) {
	os.Setenv("WWS_ENCRYPTION_KEY", "test-key")
	InitEncryption()

	_, err := Decrypt("invalid-base64!!!")
	if err == nil {
		t.Error("expected error for invalid base64")
	}
}

func TestDecryptTooShort(t *testing.T) {
	os.Setenv("WWS_ENCRYPTION_KEY", "test-key")
	InitEncryption()

	_, err := Decrypt("YWJj") // "abc" in base64
	if err == nil {
		t.Error("expected error for ciphertext too short")
	}
}

func TestDecryptWithWrongKey(t *testing.T) {
	os.Setenv("WWS_ENCRYPTION_KEY", "original-key")
	InitEncryption()

	plaintext := "secret-token"
	encrypted, _ := Encrypt(plaintext)

	os.Setenv("WWS_ENCRYPTION_KEY", "different-key")
	InitEncryption()

	_, err := Decrypt(encrypted)
	if err == nil {
		t.Error("expected error when decrypting with wrong key")
	}
}

func TestGetKeyVersion(t *testing.T) {
	os.Setenv("WWS_ENCRYPTION_KEY", "test-key")
	InitEncryption()

	version := GetKeyVersion()
	if version != KeyVersion {
		t.Errorf("expected key version %d, got %d", KeyVersion, version)
	}
}

func TestEncryptEmptyString(t *testing.T) {
	os.Setenv("WWS_ENCRYPTION_KEY", "test-key")
	InitEncryption()

	encrypted, err := Encrypt("")
	if err != nil {
		t.Fatalf("Encrypt failed for empty string: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != "" {
		t.Error("decrypted empty string should be empty")
	}
}

func TestEncryptLongToken(t *testing.T) {
	os.Setenv("WWS_ENCRYPTION_KEY", "test-key-long")
	InitEncryption()

	longToken := "ghp_" + "a" + "b" + "c" + "d" + "e" + "f" + "g" + "h" + "i" + "j" + "k" + "l" + "m" + "n" + "o" + "p" + "q" + "r" + "s" + "t" + "u" + "v" + "w" + "x" + "y" + "z" + "0" + "1" + "2" + "3" + "4" + "5" + "6" + "7" + "8" + "9"

	encrypted, err := Encrypt(longToken)
	if err != nil {
		t.Fatalf("Encrypt failed for long token: %v", err)
	}

	decrypted, err := Decrypt(encrypted)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != longToken {
		t.Errorf("decrypted '%s' does not match original '%s'", decrypted, longToken)
	}
}
