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

type serviceFilesFlag map[string]string

func (f *serviceFilesFlag) String() string { return fmt.Sprint(map[string]string(*f)) }
func (f *serviceFilesFlag) Set(value string) error {
	parts := strings.SplitN(value, "=", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return fmt.Errorf("expected platform=path")
	}
	if *f == nil {
		*f = map[string]string{}
	}
	(*f)[strings.ToLower(strings.TrimSpace(parts[0]))] = strings.TrimSpace(parts[1])
	return nil
}

func main() {
	controller := flag.String("controller", "", "Fabric Controller base URL")
	edge := flag.String("edge", "", "edge id")
	tokenFile := flag.String("token-file", "", "edge heartbeat token file")
	interval := flag.Duration("interval", 15*time.Second, "heartbeat interval")
	version := flag.String("version", "fabric-final", "edge software version")
	capacity := flag.Int("capacity", 128, "declared active-user capacity")
	activeUsersFile := flag.String("active-users-file", "", "optional file containing active user count")
	var serviceFiles serviceFilesFlag
	flag.Var(&serviceFiles, "service-file", "service call file: platform=path (repeatable)")
	flag.Parse()
	if *controller == "" || *edge == "" || *tokenFile == "" || len(serviceFiles) == 0 {
		log.Fatal("--controller, --edge, --token-file and at least one --service-file are required")
	}
	tokenRaw, err := os.ReadFile(*tokenFile)
	if err != nil {
		log.Fatal(err)
	}
	token := strings.TrimSpace(string(tokenRaw))
	client := &http.Client{Timeout: 8 * time.Second}
	for {
		calls := map[string]string{}
		for platform, path := range serviceFiles {
			if value := firstLine(path); value != "" {
				calls[platform] = value
			}
		}
		active := readIntFile(*activeUsersFile)
		load := 0.0
		if *capacity > 0 {
			load = float64(active) / float64(*capacity)
			if load > 1 {
				load = 1
			}
		}
		body, _ := json.Marshal(map[string]any{"serviceCalls": calls, "load": load, "activeUsers": active, "capacity": *capacity, "version": *version})
		url := strings.TrimRight(*controller, "/") + "/v1/edges/" + *edge + "/heartbeat"
		req, _ := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("heartbeat: %v", err)
		} else {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode/100 != 2 {
				log.Printf("heartbeat HTTP %d", resp.StatusCode)
			}
		}
		time.Sleep(*interval)
	}
}

func firstLine(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(raw), "\n") {
		if value := strings.TrimSpace(line); value != "" {
			return value
		}
	}
	return ""
}
func readIntFile(path string) int {
	if path == "" {
		return 0
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	var n int
	_, _ = fmt.Sscanf(strings.TrimSpace(string(raw)), "%d", &n)
	if n < 0 {
		return 0
	}
	return n
}
