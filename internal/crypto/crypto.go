package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/scrypt"
)

const (
	saltSize  = 32
	keySize   = 32 // AES-256
	hashSize  = 32 // Separate size for password hash to domain-separate from key
	nonceSize = 12 // GCM standard nonce size
	scryptN   = 32768
	scryptR   = 8
	scryptP   = 1
)

// DeriveKey derives an encryption key from a password using scrypt.
// Password is accepted as []byte so the caller can zero it after use.
func DeriveKey(password []byte, salt []byte) ([]byte, error) {
	return scrypt.Key(password, salt, scryptN, scryptR, scryptP, keySize)
}

// GenerateSalt generates a random salt
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, saltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// Encrypt encrypts data using AES-256-GCM
func Encrypt(plaintext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

// Decrypt decrypts data using AES-256-GCM
func Decrypt(ciphertext []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	if len(ciphertext) < gcm.NonceSize() {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

// HashPassword creates a domain-separated scrypt hash of the password for verification.
// Uses hashSize (different from keySize) to ensure the hash output differs from DeriveKey
// even with the same password and salt, preventing a single scrypt call from producing both.
// Password is accepted as []byte so the caller can zero it after use.
func HashPassword(password []byte, salt []byte) (string, error) {
	hash, err := scrypt.Key(password, salt, scryptN, scryptR, scryptP, keySize+hashSize)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	// Use only the second half (domain separation from DeriveKey which uses first keySize bytes)
	return base64.StdEncoding.EncodeToString(hash[keySize:]), nil
}

// ConstantTimeHashEqual compares two base64-encoded hashes in constant time.
func ConstantTimeHashEqual(a, b string) bool {
	aBytes, aErr := base64.StdEncoding.DecodeString(a)
	bBytes, bErr := base64.StdEncoding.DecodeString(b)
	if aErr != nil || bErr != nil {
		return false
	}
	return subtle.ConstantTimeCompare(aBytes, bBytes) == 1
}
