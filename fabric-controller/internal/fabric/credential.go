package fabric

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

func NormalizeUser(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func LoadFabricKey(path string) ([]byte, error) {
	raw, err := osReadFile(path)
	if err != nil {
		return nil, err
	}
	key, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("decode Fabric auth key: %w", err)
	}
	if len(key) < 32 {
		return nil, fmt.Errorf("Fabric auth key must contain at least 32 bytes")
	}
	return key, nil
}

var osReadFile = func(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func StaticCredential(key []byte, user string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("vconnect-fabric/v1/user/"))
	_, _ = mac.Write([]byte(NormalizeUser(user)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func VerifyCredential(key []byte, user, credential string, now time.Time) bool {
	user = NormalizeUser(user)
	credential = strings.TrimSpace(credential)
	if user == "" || credential == "" || len(key) < 32 {
		return false
	}
	if strings.HasPrefix(credential, "ft1.") {
		parts := strings.Split(credential, ".")
		if len(parts) != 4 {
			return false
		}
		exp, err := strconv.ParseInt(parts[1], 10, 64)
		if err != nil || exp <= now.Unix() || exp > now.Add(24*time.Hour).Unix()+30 {
			return false
		}
		expected := tokenSignature(key, user, exp, parts[2])
		return len(expected) == len(parts[3]) && subtle.ConstantTimeCompare([]byte(expected), []byte(parts[3])) == 1
	}
	expected := StaticCredential(key, user)
	return len(expected) == len(credential) && subtle.ConstantTimeCompare([]byte(expected), []byte(credential)) == 1
}

func IssueToken(key []byte, user string, ttl time.Duration) (string, error) {
	user = NormalizeUser(user)
	if user == "" {
		return "", fmt.Errorf("user is required")
	}
	if ttl <= 0 || ttl > 24*time.Hour {
		return "", fmt.Errorf("token TTL must be in (0,24h]")
	}
	nonceRaw := make([]byte, 16)
	if _, err := rand.Read(nonceRaw); err != nil {
		return "", err
	}
	nonce := base64.RawURLEncoding.EncodeToString(nonceRaw)
	exp := time.Now().Add(ttl).Unix()
	return fmt.Sprintf("ft1.%d.%s.%s", exp, nonce, tokenSignature(key, user, exp, nonce)), nil
}

func tokenSignature(key []byte, user string, exp int64, nonce string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("vconnect-fabric/v1/token|"))
	_, _ = mac.Write([]byte(NormalizeUser(user)))
	_, _ = mac.Write([]byte("|" + strconv.FormatInt(exp, 10) + "|" + nonce))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
