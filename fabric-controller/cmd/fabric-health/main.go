package main

import (
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	endpoint := flag.String("url", "http://127.0.0.1:47800/healthz", "health endpoint")
	flag.Parse()
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(*endpoint)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "health HTTP %d\n", resp.StatusCode)
		os.Exit(1)
	}
}
