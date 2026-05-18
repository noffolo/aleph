package main

import (
	"context"
	"embed"
	"flag"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"strconv"
	"syscall"
	"time"

	"github.com/ff3300/aleph-v2/internal/app"
	"github.com/ff3300/aleph-v2/internal/config"
	"github.com/ff3300/aleph-v2/internal/routes"
)

func init() {
	// GOMEMLIMIT — soft memory cap for the Go GC.
	// When set, the GC is more aggressive at keeping heap below this value,
	// reducing the risk of OOM kills in production (B15).
	if limit := os.Getenv("GOMEMLIMIT"); limit != "" {
		v, err := strconv.ParseInt(limit, 10, 64)
		if err == nil && v > 0 {
			debug.SetMemoryLimit(v)
			log.Printf("[Aleph] GOMEMLIMIT set to %d bytes", v)
		} else {
			log.Printf("[Aleph] Invalid GOMEMLIMIT value %q, ignoring", limit)
		}
	}
}

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

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		if err := aleph.Serve(*port); err != nil {
			log.Printf("[Aleph] Serve exited: %v", err)
		}
		stop <- os.Interrupt
	}()

	<-stop

	log.Println("[Aleph] Shutting down gracefully...")

	routes.SetDraining(true)
	time.Sleep(2 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := aleph.Close(ctx); err != nil {
		log.Printf("[Aleph] Shutdown error: %v", err)
	}
}
