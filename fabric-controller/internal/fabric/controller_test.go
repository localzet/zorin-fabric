package fabric

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestControllerHeartbeatAndBootstrap(t *testing.T) {
	remnawave := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/api/users/by-short-uuid/") {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Errorf("unexpected auth header: %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"response": map[string]any{
				"shortUuid": "user01",
				"vlessUuid": "11111111-1111-1111-1111-111111111111",
				"status":    "ACTIVE",
				"username":  "tester",
			},
		})
	}))
	defer remnawave.Close()

	dir := t.TempDir()
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i + 1)
	}
	keyFile := filepath.Join(dir, "fabric.key")
	if err := os.WriteFile(keyFile, []byte(base64.StdEncoding.EncodeToString(key)), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg := Config{
		FabricAuthKeyFile: keyFile,
		Remnawave:         RemnawaveConfig{BaseURL: remnawave.URL, APIToken: "token", LookupMode: "short-uuid"},
		Edges: []EdgeConfig{
			{ID: "ru-a", Name: "A", Region: "ru", Enabled: true, Priority: 10, HeartbeatToken: "edge-a", StaleAfter: "90s"},
			{ID: "ru-b", Name: "B", Region: "ru", Enabled: true, Priority: 20, HeartbeatToken: "edge-b", StaleAfter: "90s"},
		},
	}
	controller, err := NewController(cfg)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(controller.Handler())
	defer server.Close()

	heartbeat := func(edge, token, link string, load float64) {
		body := strings.NewReader(`{"serviceCalls":{"telemost":"` + link + `"},"load":` + formatFloat(load) + `}`)
		req, _ := http.NewRequest(http.MethodPost, server.URL+"/v1/edges/"+edge+"/heartbeat", body)
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("heartbeat %s: HTTP %d", edge, resp.StatusCode)
		}
	}
	heartbeat("ru-a", "edge-a", "https://telemost.example/a", 0.1)
	heartbeat("ru-b", "edge-b", "https://telemost.example/b", 0.9)

	credential := StaticCredential(key, "user01")
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/v1/bootstrap?user=user01", nil)
	req.Header.Set("X-Fabric-Credential", credential)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bootstrap HTTP %d", resp.StatusCode)
	}
	var payload struct {
		Edges []struct {
			ID     string `json:"id"`
			Online bool   `json:"online"`
		} `json:"edges"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Edges) != 2 || payload.Edges[0].ID != "ru-b" || !payload.Edges[0].Online {
		t.Fatalf("unexpected edge order/status: %+v", payload.Edges)
	}
}

func formatFloat(value float64) string {
	raw, _ := json.Marshal(value)
	return string(raw)
}
