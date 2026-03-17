package main

import (
	"log"
	"net/http"
	"time"

	"read_article/backend/internal/api"
	"read_article/backend/internal/config"
	"read_article/backend/internal/inference"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	client, err := inference.NewClient(cfg)
	if err != nil {
		log.Fatalf("create inference client: %v", err)
	}

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           api.NewServer(cfg, client).Routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	log.Printf("read article server listening on :%s", cfg.ServerPort)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("listen and serve: %v", err)
	}
}
