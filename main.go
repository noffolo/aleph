package main

import (
	"context"
	"embed"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ff3300/aleph-v2/internal/app"
	"github.com/ff3300/aleph-v2/internal/config"
)

//go:embed dist/*
var frontendFS embed.FS

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if cfg.EncryptionKey == nil {
		log.Println("WARNING: KEY_ENCRYPTION_KEY not set — API keys stored in PLAINTEXT")
	}

	port := flag.Int("port", cfg.Port, "Port to listen on")
	flag.Parse()

	aleph, err := app.NewAlephApp(cfg, frontendFS)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		if err := aleph.Serve(*port); err != nil {
			log.Fatal(err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("[Aleph] Shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := aleph.Close(ctx); err != nil {
		log.Printf("[Aleph] Shutdown error: %v", err)
	}
}
