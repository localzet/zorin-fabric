package fabric

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

type Config struct {
	Listen            string          `json:"listen"`
	FabricAuthKeyFile string          `json:"fabricAuthKeyFile"`
	Remnawave         RemnawaveConfig `json:"remnawave"`
	Trust             TrustConfig     `json:"trust"`
	Edges             []EdgeConfig    `json:"edges"`
	BootstrapCacheTTL string          `json:"bootstrapCacheTtl,omitempty"`
}

type RemnawaveConfig struct {
	BaseURL      string `json:"baseUrl"`
	APIToken     string `json:"apiToken,omitempty"`
	APITokenFile string `json:"apiTokenFile,omitempty"`
	LookupMode   string `json:"lookupMode,omitempty"`
}

type TrustConfig struct {
	Enabled      bool   `json:"enabled"`
	RegistryFile string `json:"registryFile,omitempty"`
	TokenTTL     string `json:"tokenTtl,omitempty"`
}

type EdgeConfig struct {
	ID                 string            `json:"id"`
	Region             string            `json:"region"`
	Name               string            `json:"name,omitempty"`
	Enabled            bool              `json:"enabled"`
	Priority           int               `json:"priority,omitempty"`
	HeartbeatToken     string            `json:"heartbeatToken,omitempty"`
	HeartbeatTokenFile string            `json:"heartbeatTokenFile,omitempty"`
	StaleAfter         string            `json:"staleAfter,omitempty"`
	ServiceCalls       map[string]string `json:"serviceCalls,omitempty"`
}

func LoadConfig(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	if strings.TrimSpace(cfg.Listen) == "" {
		cfg.Listen = "127.0.0.1:47800"
	}
	if strings.TrimSpace(cfg.FabricAuthKeyFile) == "" {
		return Config{}, fmt.Errorf("fabricAuthKeyFile is required")
	}
	if strings.TrimSpace(cfg.Remnawave.BaseURL) == "" {
		return Config{}, fmt.Errorf("remnawave.baseUrl is required")
	}
	if cfg.Remnawave.APIToken == "" && cfg.Remnawave.APITokenFile != "" {
		data, err := os.ReadFile(cfg.Remnawave.APITokenFile)
		if err != nil {
			return Config{}, fmt.Errorf("read Remnawave API token: %w", err)
		}
		cfg.Remnawave.APIToken = strings.TrimSpace(string(data))
	}
	if cfg.Remnawave.APIToken == "" {
		return Config{}, fmt.Errorf("remnawave.apiToken or apiTokenFile is required")
	}
	seen := map[string]bool{}
	for i := range cfg.Edges {
		edge := &cfg.Edges[i]
		edge.ID = strings.TrimSpace(edge.ID)
		if edge.ID == "" {
			return Config{}, fmt.Errorf("edges[%d].id is required", i)
		}
		if seen[edge.ID] {
			return Config{}, fmt.Errorf("duplicate edge id %q", edge.ID)
		}
		seen[edge.ID] = true
		if edge.HeartbeatToken == "" && edge.HeartbeatTokenFile != "" {
			data, err := os.ReadFile(edge.HeartbeatTokenFile)
			if err != nil {
				return Config{}, fmt.Errorf("read edge token %s: %w", edge.ID, err)
			}
			edge.HeartbeatToken = strings.TrimSpace(string(data))
		}
		if edge.StaleAfter == "" {
			edge.StaleAfter = "90s"
		}
		if _, err := time.ParseDuration(edge.StaleAfter); err != nil {
			return Config{}, fmt.Errorf("edge %s staleAfter: %w", edge.ID, err)
		}
	}
	return cfg, nil
}
