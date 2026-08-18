package fabric

import (
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

type EdgeHeartbeat struct {
	ServiceCalls map[string]string `json:"serviceCalls,omitempty"`
	Load         float64           `json:"load,omitempty"`
	ActiveUsers  int               `json:"activeUsers,omitempty"`
	Capacity     int               `json:"capacity,omitempty"`
	Version      string            `json:"version,omitempty"`
}

type edgeState struct {
	Config    EdgeConfig
	Heartbeat EdgeHeartbeat
	LastSeen  time.Time
}

type Controller struct {
	cfg         Config
	key         []byte
	remnawave   *RemnawaveClient
	trust       TrustRegistry
	trustTTL    time.Duration
	mu          sync.RWMutex
	edges       map[string]*edgeState
	trustNonces map[string]int64
}

func NewController(cfg Config) (*Controller, error) {
	key, err := LoadFabricKey(cfg.FabricAuthKeyFile)
	if err != nil {
		return nil, err
	}
	controller := &Controller{
		cfg: cfg, key: key, remnawave: NewRemnawaveClient(cfg.Remnawave), edges: map[string]*edgeState{}, trustNonces: map[string]int64{}, trustTTL: time.Hour,
	}
	if cfg.Trust.Enabled {
		if strings.TrimSpace(cfg.Trust.RegistryFile) == "" {
			return nil, fmt.Errorf("trust.registryFile is required when Trust exchange is enabled")
		}
		controller.trust, err = LoadTrustRegistry(cfg.Trust.RegistryFile)
		if err != nil {
			return nil, fmt.Errorf("load Trust registry: %w", err)
		}
		if cfg.Trust.TokenTTL != "" {
			controller.trustTTL, err = time.ParseDuration(cfg.Trust.TokenTTL)
			if err != nil {
				return nil, fmt.Errorf("trust.tokenTtl: %w", err)
			}
		}
		if controller.trustTTL <= 0 || controller.trustTTL > 24*time.Hour {
			return nil, fmt.Errorf("trust.tokenTtl must be in (0,24h]")
		}
	}
	for _, edge := range cfg.Edges {
		state := &edgeState{Config: edge, Heartbeat: EdgeHeartbeat{ServiceCalls: cloneMap(edge.ServiceCalls)}}
		if len(edge.ServiceCalls) > 0 {
			state.LastSeen = time.Now()
		}
		controller.edges[edge.ID] = state
	}
	return controller, nil
}

func (c *Controller) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", c.health)
	mux.HandleFunc("GET /v1/bootstrap", c.bootstrap)
	mux.HandleFunc("POST /v1/edges/{edge}/heartbeat", c.heartbeat)
	mux.HandleFunc("POST /v1/trust/exchange", c.trustExchange)
	return withJSONLogging(mux)
}

func (c *Controller) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "time": time.Now().UTC().Format(time.RFC3339)})
}

func (c *Controller) bootstrap(w http.ResponseWriter, r *http.Request) {
	user := NormalizeUser(firstNonEmpty(r.Header.Get("X-Fabric-User"), r.URL.Query().Get("user")))
	credential := strings.TrimSpace(firstNonEmpty(r.Header.Get("X-Fabric-Credential"), bearerValue(r.Header.Get("Authorization"))))
	if !VerifyCredential(c.key, user, credential, time.Now()) {
		writeError(w, http.StatusUnauthorized, "invalid Fabric credential")
		return
	}
	if _, err := c.remnawave.ActiveUser(r.Context(), user); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}

	now := time.Now()
	c.mu.RLock()
	items := make([]map[string]any, 0, len(c.edges))
	for _, state := range c.edges {
		if !state.Config.Enabled {
			continue
		}
		staleAfter, _ := time.ParseDuration(state.Config.StaleAfter)
		online := !state.LastSeen.IsZero() && now.Sub(state.LastSeen) <= staleAfter
		calls := cloneMap(state.Heartbeat.ServiceCalls)
		if len(calls) == 0 {
			calls = cloneMap(state.Config.ServiceCalls)
		}
		if len(calls) == 0 {
			online = false
		}
		items = append(items, map[string]any{
			"id": state.Config.ID, "name": state.Config.Name, "region": state.Config.Region,
			"priority": state.Config.Priority, "online": online, "lastSeen": state.LastSeen.UTC().Format(time.RFC3339),
			"load": state.Heartbeat.Load, "activeUsers": state.Heartbeat.ActiveUsers, "capacity": state.Heartbeat.Capacity,
			"serviceCalls": calls, "version": state.Heartbeat.Version,
		})
	}
	c.mu.RUnlock()
	sort.SliceStable(items, func(i, j int) bool {
		aOnline, _ := items[i]["online"].(bool)
		bOnline, _ := items[j]["online"].(bool)
		if aOnline != bOnline {
			return aOnline
		}
		ap, _ := items[i]["priority"].(int)
		bp, _ := items[j]["priority"].(int)
		if ap != bp {
			return ap > bp
		}
		al, _ := items[i]["load"].(float64)
		bl, _ := items[j]["load"].(float64)
		return al < bl
	})
	writeJSON(w, http.StatusOK, map[string]any{
		"schemaVersion": 1, "user": user, "issuedAt": now.UTC().Format(time.RFC3339), "edges": items,
	})
}

func (c *Controller) heartbeat(w http.ResponseWriter, r *http.Request) {
	edgeID := strings.TrimSpace(r.PathValue("edge"))
	c.mu.Lock()
	state := c.edges[edgeID]
	c.mu.Unlock()
	if state == nil {
		writeError(w, http.StatusNotFound, "unknown edge")
		return
	}
	provided := bearerValue(r.Header.Get("Authorization"))
	if provided == "" || len(provided) != len(state.Config.HeartbeatToken) || subtle.ConstantTimeCompare([]byte(provided), []byte(state.Config.HeartbeatToken)) != 1 {
		writeError(w, http.StatusUnauthorized, "invalid edge token")
		return
	}
	var heartbeat EdgeHeartbeat
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&heartbeat); err != nil {
		writeError(w, http.StatusBadRequest, "invalid heartbeat")
		return
	}
	if heartbeat.Load < 0 {
		heartbeat.Load = 0
	}
	if heartbeat.Load > 1 {
		heartbeat.Load = 1
	}
	normalizedCalls := make(map[string]string, len(heartbeat.ServiceCalls))
	for platform, link := range heartbeat.ServiceCalls {
		platform = strings.ToLower(strings.TrimSpace(platform))
		link = strings.TrimSpace(link)
		if platform == "" || link == "" {
			continue
		}
		switch platform {
		case "telemost", "vk":
			normalizedCalls[platform] = link
		}
	}
	heartbeat.ServiceCalls = normalizedCalls
	c.mu.Lock()
	state.Heartbeat = heartbeat
	state.LastSeen = time.Now()
	c.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "edge": edgeID})
}

type trustExchangeRequest struct {
	User  string     `json:"user"`
	Proof OwnerProof `json:"proof"`
}

func (c *Controller) trustExchange(w http.ResponseWriter, r *http.Request) {
	if !c.cfg.Trust.Enabled {
		writeError(w, http.StatusNotFound, "Trust exchange is disabled")
		return
	}
	var request trustExchangeRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request")
		return
	}
	request.User = NormalizeUser(request.User)
	if err := VerifyOwnerProof(request.Proof, c.trust, request.User, time.Now()); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if _, err := c.remnawave.ActiveUser(r.Context(), request.User); err != nil {
		writeError(w, http.StatusForbidden, err.Error())
		return
	}
	if !c.consumeTrustNonce(request.Proof, time.Now()) {
		writeError(w, http.StatusConflict, "owner proof was already consumed")
		return
	}
	token, err := IssueToken(c.key, request.User, c.trustTTL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "could not issue token")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"schemaVersion": 1, "user": request.User, "credential": token, "expiresInSeconds": int(c.trustTTL.Seconds()),
	})
}

func (c *Controller) consumeTrustNonce(proof OwnerProof, now time.Time) bool {
	key := strings.ToUpper(strings.TrimSpace(proof.PhoneFingerprint)) + "|" + strings.TrimSpace(proof.Nonce)
	if key == "|" {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	for nonce, expires := range c.trustNonces {
		if expires <= now.Unix() {
			delete(c.trustNonces, nonce)
		}
	}
	if _, exists := c.trustNonces[key]; exists {
		return false
	}
	c.trustNonces[key] = proof.Expires
	return true
}

func cloneMap(input map[string]string) map[string]string {
	out := make(map[string]string, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func bearerValue(value string) string {
	value = strings.TrimSpace(value)
	for _, prefix := range []string{"Bearer ", "Fabric "} {
		if strings.HasPrefix(value, prefix) {
			return strings.TrimSpace(strings.TrimPrefix(value, prefix))
		}
	}
	return value
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]any{"error": message})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func withJSONLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		started := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(started).Round(time.Millisecond))
	})
}
