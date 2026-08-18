package controlplane

import (
	"context"
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDeriveFabricCredentialStable(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	first := DeriveFabricCredential(key, "abc")
	second := DeriveFabricCredential(key, "abc")
	if first == "" || first != second {
		t.Fatalf("credential must be stable")
	}
	if first == DeriveFabricCredential(key, "other") {
		t.Fatalf("credentials for different users must differ")
	}
}

func TestRemnawaveAuthorizerActiveUser(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer api-token" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if r.URL.Path != "/api/users/by-short-uuid/u123" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"status":"ACTIVE"}}`))
	}))
	defer server.Close()

	authorizer, err := NewRemnawaveAuthorizer(RemnawaveAuthorizerConfig{
		BaseURL:       server.URL,
		APIToken:      "api-token",
		LookupMode:    RemnawaveLookupShortUUID,
		CredentialKey: key,
		CacheTTL:      time.Second,
		StaleGrace:    time.Second,
	})
	if err != nil {
		t.Fatal(err)
	}

	credential := DeriveFabricCredential(key, "u123")
	userID, err := authorizer.AuthorizeUser(context.Background(), "u123", credential)
	if err != nil {
		t.Fatal(err)
	}
	if userID != "u123" {
		t.Fatalf("unexpected user id %q", userID)
	}
}

func TestRemnawaveAuthorizerRejectsDisabledUser(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"response":{"status":"DISABLED"}}`))
	}))
	defer server.Close()

	authorizer, err := NewRemnawaveAuthorizer(RemnawaveAuthorizerConfig{
		BaseURL:       server.URL,
		APIToken:      "api-token",
		CredentialKey: key,
	})
	if err != nil {
		t.Fatal(err)
	}
	credential := DeriveFabricCredential(key, "u123")
	if _, err := authorizer.AuthorizeUser(context.Background(), "u123", credential); err == nil {
		t.Fatal("disabled user must be rejected")
	}
}

func TestFabricCredentialKeyFromBase64(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	decoded, err := FabricCredentialKeyFromBase64(base64.StdEncoding.EncodeToString(key))
	if err != nil {
		t.Fatal(err)
	}
	if string(decoded) != string(key) {
		t.Fatal("decoded key mismatch")
	}
}
