package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type response struct {
	Edges []struct {
		ID           string            `json:"id"`
		Online       bool              `json:"online"`
		ServiceCalls map[string]string `json:"serviceCalls"`
	} `json:"edges"`
}

func main() {
	controller := flag.String("controller", "", "Fabric Controller base URL")
	user := flag.String("user", "", "Remnawave user id")
	credential := flag.String("credential", "", "Fabric credential/token")
	platform := flag.String("platform", "telemost", "telemost or vk")
	jsonOut := flag.Bool("json", false, "print full bootstrap JSON")
	flag.Parse()
	if *controller == "" || *user == "" || *credential == "" {
		log.Fatal("--controller, --user and --credential are required")
	}
	u := strings.TrimRight(*controller, "/") + "/v1/bootstrap?user=" + url.QueryEscape(*user)
	req, _ := http.NewRequest(http.MethodGet, u, nil)
	req.Header.Set("X-Fabric-Credential", *credential)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		log.Fatalf("bootstrap HTTP %d", resp.StatusCode)
	}
	var raw json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		log.Fatal(err)
	}
	if *jsonOut {
		var pretty any
		_ = json.Unmarshal(raw, &pretty)
		out, _ := json.MarshalIndent(pretty, "", "  ")
		fmt.Println(string(out))
		return
	}
	var data response
	if err := json.Unmarshal(raw, &data); err != nil {
		log.Fatal(err)
	}
	for _, edge := range data.Edges {
		if edge.Online {
			if link := strings.TrimSpace(edge.ServiceCalls[strings.ToLower(*platform)]); link != "" {
				fmt.Printf("%s\t%s\n", edge.ID, link)
				return
			}
		}
	}
	log.Fatal("no online edge with requested platform")
}
