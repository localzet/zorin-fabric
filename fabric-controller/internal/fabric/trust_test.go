package fabric

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"strconv"
	"testing"
	"time"
)

func TestOwnerProofAndReplayCache(t *testing.T) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	der, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	phoneFP := fingerprint(der)
	now := time.Now().Unix()
	proof := OwnerProof{
		Version:              "ZOWNER/1",
		Action:               "fabric.session.issue",
		Resource:             "fabric:user:user01",
		HostFingerprint:      "HOST",
		PhoneFingerprint:     phoneFP,
		PhonePublicKeyDERHex: hex.EncodeToString(der),
		Nonce:                "nonce-123",
		Issued:               now,
		Expires:              now + 60,
	}
	actionHex := hex.EncodeToString([]byte(proof.Action))
	resourceHex := hex.EncodeToString([]byte(proof.Resource))
	message := "ZTRUST/2|OWNER_PROOF|" + proof.HostFingerprint + "|" + proof.PhoneFingerprint + "|" + actionHex + "|" + resourceHex + "|" + proof.Nonce + "|" + strconv.FormatInt(proof.Issued, 10) + "|" + strconv.FormatInt(proof.Expires, 10)
	digest := sha256.Sum256([]byte(message))
	signature, err := ecdsa.SignASN1(rand.Reader, privateKey, digest[:])
	if err != nil {
		t.Fatal(err)
	}
	proof.SignatureDERHex = hex.EncodeToString(signature)

	registry := TrustRegistry{
		TrustedPhones: map[string]string{phoneFP: proof.PhonePublicKeyDERHex},
		Bindings:      map[string]string{phoneFP: "user01"},
	}
	if err := VerifyOwnerProof(proof, registry, "user01", time.Unix(now, 0)); err != nil {
		t.Fatalf("valid proof rejected: %v", err)
	}
	controller := &Controller{trustNonces: map[string]int64{}}
	if !controller.consumeTrustNonce(proof, time.Unix(now, 0)) {
		t.Fatal("first proof use must be accepted")
	}
	if controller.consumeTrustNonce(proof, time.Unix(now+1, 0)) {
		t.Fatal("replayed proof must be rejected")
	}
}
