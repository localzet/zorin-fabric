package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"zorin-fabric/controller/internal/fabric"
)

func main() {
	configPath := flag.String("config", "fabric-controller.json", "path to controller config")
	flag.Parse()
	cfg, err := fabric.LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}
	controller, err := fabric.NewController(cfg)
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{Addr: cfg.Listen, Handler: controller.Handler(), ReadHeaderTimeout: 5 * time.Second, IdleTimeout: 60 * time.Second}
	log.Printf("Zorin Fabric Controller listening on %s", cfg.Listen)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Println(err)
		os.Exit(1)
	}
}
