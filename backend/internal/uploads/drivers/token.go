package drivers

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// GenerateToken creates an HMAC-SHA256 token signing multiple constraints.
func GenerateToken(key, secret string, expiresAt int64, contentType string, maxSizeBytes int64) string {
	payload := fmt.Sprintf("%s:%d:%s:%d", key, expiresAt, contentType, maxSizeBytes)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyToken checks if a provided token matches the expected signature for given constraints.
func VerifyToken(key, token, secret string, expiresAt int64, contentType string, maxSizeBytes int64) bool {
	if token == "" || secret == "" {
		return false
	}
	expected := GenerateToken(key, secret, expiresAt, contentType, maxSizeBytes)
	return hmac.Equal([]byte(token), []byte(expected))
}

// GenerateDownloadToken creates an HMAC-SHA256 token specifically for download links (only signs key and expiration).
func GenerateDownloadToken(key, secret string, expiresAt int64) string {
	payload := fmt.Sprintf("%s:%d", key, expiresAt)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// VerifyDownloadToken checks if a provided download token matches the expected signature.
func VerifyDownloadToken(key, token, secret string, expiresAt int64) bool {
	if token == "" || secret == "" {
		return false
	}
	expected := GenerateDownloadToken(key, secret, expiresAt)
	return hmac.Equal([]byte(token), []byte(expected))
}
