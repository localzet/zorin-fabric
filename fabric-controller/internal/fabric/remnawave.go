package fabric

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type RemnawaveUser struct {
	ShortUUID string `json:"shortUuid"`
	VLESSUUID string `json:"vlessUuid"`
	Username  string `json:"username"`
	Status    string `json:"status"`
}

type cachedUser struct {
	User      RemnawaveUser
	CheckedAt time.Time
}

type RemnawaveClient struct {
	baseURL string
	token   string
	lookup  string
	http    *http.Client
	mu      sync.Mutex
	cache   map[string]cachedUser
}

func NewRemnawaveClient(cfg RemnawaveConfig) *RemnawaveClient {
	lookup := strings.TrimSpace(cfg.LookupMode)
	if lookup == "" {
		lookup = "short-uuid"
	}
	return &RemnawaveClient{
		baseURL: strings.TrimRight(cfg.BaseURL, "/"), token: cfg.APIToken, lookup: lookup,
		http: &http.Client{Timeout: 8 * time.Second}, cache: map[string]cachedUser{},
	}
}

func (c *RemnawaveClient) ActiveUser(ctx context.Context, userID string) (RemnawaveUser, error) {
	userID = NormalizeUser(userID)
	c.mu.Lock()
	cached, ok := c.cache[userID]
	c.mu.Unlock()
	if ok && time.Since(cached.CheckedAt) < 30*time.Second {
		if strings.EqualFold(cached.User.Status, "ACTIVE") {
			return cached.User, nil
		}
		return RemnawaveUser{}, fmt.Errorf("Remnawave user is %s", cached.User.Status)
	}

	endpoint, err := c.endpoint(userID)
	if err != nil {
		return RemnawaveUser{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return RemnawaveUser{}, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return RemnawaveUser{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return RemnawaveUser{}, err
	}
	if resp.StatusCode != http.StatusOK {
		return RemnawaveUser{}, fmt.Errorf("Remnawave HTTP %d", resp.StatusCode)
	}
	var payload any
	if err := json.Unmarshal(body, &payload); err != nil {
		return RemnawaveUser{}, err
	}
	user := RemnawaveUser{}
	user.ShortUUID, _ = findString(payload, "shortUuid")
	user.VLESSUUID, _ = findString(payload, "vlessUuid")
	user.Username, _ = findString(payload, "username")
	user.Status, _ = findString(payload, "status")
	if user.ShortUUID == "" {
		user.ShortUUID = userID
	}
	c.mu.Lock()
	c.cache[userID] = cachedUser{User: user, CheckedAt: time.Now()}
	c.mu.Unlock()
	if !strings.EqualFold(user.Status, "ACTIVE") {
		return RemnawaveUser{}, fmt.Errorf("Remnawave user is %s", user.Status)
	}
	return user, nil
}

func (c *RemnawaveClient) endpoint(user string) (string, error) {
	escaped := url.PathEscape(user)
	switch c.lookup {
	case "short-uuid":
		return c.baseURL + "/api/users/by-short-uuid/" + escaped, nil
	case "username":
		return c.baseURL + "/api/users/by-username/" + escaped, nil
	case "id":
		return c.baseURL + "/api/users/" + escaped, nil
	default:
		return "", fmt.Errorf("unsupported lookup mode %q", c.lookup)
	}
}

func findString(value any, key string) (string, bool) {
	switch typed := value.(type) {
	case map[string]any:
		for k, v := range typed {
			if strings.EqualFold(k, key) {
				if text, ok := v.(string); ok {
					return text, true
				}
			}
		}
		for _, v := range typed {
			if text, ok := findString(v, key); ok {
				return text, true
			}
		}
	case []any:
		for _, v := range typed {
			if text, ok := findString(v, key); ok {
				return text, true
			}
		}
	}
	return "", false
}
