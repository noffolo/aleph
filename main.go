package main

import (
	"embed"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

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

	// Graceful Shutdown Handler
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	log.Println("[Aleph] Shutting down gracefully...")
	// Cleanup operations would go here if needed via an app.Close() method
}
