package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type request struct {
	User  string          `json:"user"`
	Proof json.RawMessage `json:"proof"`
}

type response struct {
	User             string `json:"user"`
	Credential       string `json:"credential"`
	ExpiresInSeconds int    `json:"expiresInSeconds"`
}

func main() {
	controller := flag.String("controller", "", "Fabric Controller base URL")
	user := flag.String("user", "", "Remnawave user id")
	proofFile := flag.String("proof", "", "ZOWNER/1 proof JSON file")
	provision := flag.Bool("provision", false, "print user:credential for direct client provisioning")
	flag.Parse()
	if *controller == "" || *user == "" || *proofFile == "" {
		log.Fatal("--controller, --user and --proof are required")
	}
	proofRaw, err := os.ReadFile(*proofFile)
	if err != nil {
		log.Fatal(err)
	}
	var proof json.RawMessage
	if err := json.Unmarshal(proofRaw, &proof); err != nil {
		log.Fatalf("proof JSON: %v", err)
	}
	body, _ := json.Marshal(request{User: strings.TrimSpace(*user), Proof: proof})
	endpoint := strings.TrimRight(*controller, "/") + "/v1/trust/exchange"
	req, _ := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("Trust exchange HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(raw)))
	}
	var out response
	if err := json.Unmarshal(raw, &out); err != nil {
		log.Fatal(err)
	}
	if *provision {
		fmt.Printf("%s:%s\n", out.User, out.Credential)
		return
	}
	pretty, _ := json.MarshalIndent(out, "", "  ")
	fmt.Println(string(pretty))
}
