package controlplane

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	RemnawaveLookupShortUUID = "short-uuid"
	RemnawaveLookupUsername  = "username"
	RemnawaveLookupID        = "id"
)

type RemnawaveAuthorizerConfig struct {
	BaseURL       string
	APIToken      string
	LookupMode    string
	CredentialKey []byte
	CacheTTL      time.Duration
	StaleGrace    time.Duration
	HTTPClient    *http.Client
}

type remnawaveAuthorizationCacheEntry struct {
	allowed   bool
	checkedAt time.Time
}

type RemnawaveAuthorizer struct {
	baseURL       string
	apiToken      string
	lookupMode    string
	credentialKey []byte
	cacheTTL      time.Duration
	staleGrace    time.Duration
	httpClient    *http.Client

	mu    sync.Mutex
	cache map[string]remnawaveAuthorizationCacheEntry
}

func NewRemnawaveAuthorizer(config RemnawaveAuthorizerConfig) (*RemnawaveAuthorizer, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(config.BaseURL), "/")
	if baseURL == "" {
		return nil, errors.New("controlplane: remnawave base URL is required")
	}
	if _, err := url.ParseRequestURI(baseURL); err != nil {
		return nil, fmt.Errorf("controlplane: invalid remnawave base URL: %w", err)
	}
	if strings.TrimSpace(config.APIToken) == "" {
		return nil, errors.New("controlplane: remnawave API token is required")
	}
	if len(config.CredentialKey) < 32 {
		return nil, errors.New("controlplane: fabric credential key must contain at least 32 bytes")
	}

	lookupMode := strings.TrimSpace(config.LookupMode)
	if lookupMode == "" {
		lookupMode = RemnawaveLookupShortUUID
	}
	switch lookupMode {
	case RemnawaveLookupShortUUID, RemnawaveLookupUsername, RemnawaveLookupID:
	default:
		return nil, fmt.Errorf("controlplane: unsupported remnawave lookup mode %q", lookupMode)
	}

	cacheTTL := config.CacheTTL
	if cacheTTL <= 0 {
		cacheTTL = 30 * time.Second
	}
	staleGrace := config.StaleGrace
	if staleGrace < cacheTTL {
		staleGrace = cacheTTL
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 8 * time.Second}
	}

	return &RemnawaveAuthorizer{
		baseURL:       baseURL,
		apiToken:      strings.TrimSpace(config.APIToken),
		lookupMode:    lookupMode,
		credentialKey: append([]byte(nil), config.CredentialKey...),
		cacheTTL:      cacheTTL,
		staleGrace:    staleGrace,
		httpClient:    httpClient,
		cache:         make(map[string]remnawaveAuthorizationCacheEntry),
	}, nil
}

func FabricCredentialKeyFromBase64(value string) ([]byte, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, errors.New("empty fabric credential key")
	}
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("decode fabric credential key: %w", err)
	}
	if len(key) < 32 {
		return nil, errors.New("fabric credential key must contain at least 32 bytes")
	}
	return key, nil
}

func DeriveFabricCredential(key []byte, userID string) string {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write([]byte("vconnect-fabric/v1/user/"))
	_, _ = mac.Write([]byte(normalizeUserID(userID)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (a *RemnawaveAuthorizer) AuthorizeUser(ctx context.Context, userID, credential string) (string, error) {
	userID = normalizeUserID(userID)
	credential = strings.TrimSpace(credential)
	if userID == "" || credential == "" {
		return "", errors.New("controlplane: user credential is required")
	}
	if !a.validCredential(userID, credential) {
		return "", errors.New("controlplane: user is not authorized")
	}

	now := time.Now()
	if entry, ok := a.cached(userID); ok && now.Sub(entry.checkedAt) <= a.cacheTTL {
		if entry.allowed {
			return userID, nil
		}
		return "", errors.New("controlplane: remnawave user is not active")
	}

	allowed, err := a.fetchActiveStatus(ctx, userID)
	if err != nil {
		if entry, ok := a.cached(userID); ok && entry.allowed && now.Sub(entry.checkedAt) <= a.staleGrace {
			return userID, nil
		}
		return "", fmt.Errorf("controlplane: remnawave authorization unavailable: %w", err)
	}
	a.store(userID, allowed, now)
	if !allowed {
		return "", errors.New("controlplane: remnawave user is not active")
	}
	return userID, nil
}

func (a *RemnawaveAuthorizer) validCredential(userID, credential string) bool {
	expected := DeriveFabricCredential(a.credentialKey, userID)
	if len(expected) != len(credential) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(credential)) == 1
}

func (a *RemnawaveAuthorizer) fetchActiveStatus(ctx context.Context, userID string) (bool, error) {
	endpoint, err := a.userEndpoint(userID)
	if err != nil {
		return false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+a.apiToken)
	req.Header.Set("Accept", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return false, err
	}
	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return false, fmt.Errorf("decode user response: %w", err)
	}
	status, ok := findStringField(payload, "status")
	if !ok {
		return false, errors.New("user response does not contain status")
	}
	return strings.EqualFold(status, "ACTIVE"), nil
}

func (a *RemnawaveAuthorizer) userEndpoint(userID string) (string, error) {
	escaped := url.PathEscape(userID)
	switch a.lookupMode {
	case RemnawaveLookupShortUUID:
		return a.baseURL + "/api/users/by-short-uuid/" + escaped, nil
	case RemnawaveLookupUsername:
		return a.baseURL + "/api/users/by-username/" + escaped, nil
	case RemnawaveLookupID:
		return a.baseURL + "/api/users/" + escaped, nil
	default:
		return "", fmt.Errorf("unsupported remnawave lookup mode %q", a.lookupMode)
	}
}

func (a *RemnawaveAuthorizer) cached(userID string) (remnawaveAuthorizationCacheEntry, bool) {
	a.mu.Lock()
	defer a.mu.Unlock()
	entry, ok := a.cache[userID]
	return entry, ok
}

func (a *RemnawaveAuthorizer) store(userID string, allowed bool, checkedAt time.Time) {
	a.mu.Lock()
	a.cache[userID] = remnawaveAuthorizationCacheEntry{allowed: allowed, checkedAt: checkedAt}
	a.mu.Unlock()
}

func findStringField(value any, key string) (string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		for candidateKey, candidateValue := range typed {
			if strings.EqualFold(candidateKey, key) {
				if text, ok := candidateValue.(string); ok {
					return text, true
				}
			}
		}
		for _, candidateValue := range typed {
			if text, ok := findStringField(candidateValue, key); ok {
				return text, true
			}
		}
	case []any:
		for _, candidateValue := range typed {
			if text, ok := findStringField(candidateValue, key); ok {
				return text, true
			}
		}
	}
	return "", false
}
