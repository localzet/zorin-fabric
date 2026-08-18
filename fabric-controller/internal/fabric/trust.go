package fabric

import (
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type OwnerProof struct {
	Version              string `json:"version"`
	Action               string `json:"action"`
	Resource             string `json:"resource"`
	HostFingerprint      string `json:"host_fingerprint"`
	PhoneFingerprint     string `json:"phone_fingerprint"`
	PhonePublicKeyDERHex string `json:"phone_public_key_der_hex"`
	Nonce                string `json:"nonce"`
	Issued               int64  `json:"issued_unix"`
	Expires              int64  `json:"expires_unix"`
	SignatureDERHex      string `json:"signature_der_hex"`
}

type TrustRegistry struct {
	TrustedPhones map[string]string `json:"trustedPhones"`
	Bindings      map[string]string `json:"bindings"`
}

func LoadTrustRegistry(path string) (TrustRegistry, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return TrustRegistry{}, err
	}
	var registry TrustRegistry
	if err := json.Unmarshal(raw, &registry); err != nil {
		return TrustRegistry{}, err
	}
	if registry.TrustedPhones == nil {
		registry.TrustedPhones = map[string]string{}
	}
	if registry.Bindings == nil {
		registry.Bindings = map[string]string{}
	}
	return registry, nil
}

func VerifyOwnerProof(proof OwnerProof, registry TrustRegistry, user string, now time.Time) error {
	user = NormalizeUser(user)
	if proof.Version != "ZOWNER/1" {
		return fmt.Errorf("unsupported proof version %q", proof.Version)
	}
	if proof.Action != "fabric.session.issue" || proof.Resource != "fabric:user:"+user {
		return fmt.Errorf("owner proof scope mismatch")
	}
	if proof.Expires <= now.Unix() || proof.Expires-proof.Issued > 120 || proof.Issued > now.Unix()+30 {
		return fmt.Errorf("owner proof timing invalid")
	}
	der, err := hex.DecodeString(proof.PhonePublicKeyDERHex)
	if err != nil {
		return err
	}
	canonicalPhoneFP := fingerprint(der)
	if !strings.EqualFold(canonicalPhoneFP, strings.TrimSpace(proof.PhoneFingerprint)) {
		return fmt.Errorf("phone fingerprint mismatch")
	}
	trusted, ok := registry.TrustedPhones[canonicalPhoneFP]
	if !ok {
		trusted, ok = registry.TrustedPhones[strings.TrimSpace(proof.PhoneFingerprint)]
	}
	if !ok || !strings.EqualFold(trusted, proof.PhonePublicKeyDERHex) {
		return fmt.Errorf("phone is not trusted")
	}
	boundUser := NormalizeUser(registry.Bindings[canonicalPhoneFP])
	if boundUser == "" {
		boundUser = NormalizeUser(registry.Bindings[strings.TrimSpace(proof.PhoneFingerprint)])
	}
	if boundUser == "" || boundUser != user {
		return fmt.Errorf("phone is not bound to requested Remnawave user")
	}
	parsed, err := x509.ParsePKIXPublicKey(der)
	if err != nil {
		return err
	}
	pub, ok := parsed.(*ecdsa.PublicKey)
	if !ok {
		return fmt.Errorf("phone key is not ECDSA")
	}
	sig, err := hex.DecodeString(proof.SignatureDERHex)
	if err != nil {
		return err
	}
	actionHex := hex.EncodeToString([]byte(proof.Action))
	resourceHex := hex.EncodeToString([]byte(proof.Resource))
	message := "ZTRUST/2|OWNER_PROOF|" + proof.HostFingerprint + "|" + proof.PhoneFingerprint + "|" + actionHex + "|" + resourceHex + "|" + proof.Nonce + "|" + strconv.FormatInt(proof.Issued, 10) + "|" + strconv.FormatInt(proof.Expires, 10)
	digest := sha256.Sum256([]byte(message))
	if !ecdsa.VerifyASN1(pub, digest[:], sig) {
		return fmt.Errorf("owner proof signature invalid")
	}
	return nil
}

func fingerprint(der []byte) string {
	hash := sha256.Sum256(der)
	hexFP := strings.ToUpper(hex.EncodeToString(hash[:16]))
	parts := make([]string, 0, len(hexFP)/4)
	for i := 0; i < len(hexFP); i += 4 {
		parts = append(parts, hexFP[i:i+4])
	}
	return strings.Join(parts, ":")
}
