package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/einapoli1/openclaw-go/internal/config"
	"github.com/einapoli1/openclaw-go/internal/gateway"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("openclaw-go — lightweight AI agent gateway")
		fmt.Println("Usage: openclaw-go <command>")
		fmt.Println("Commands: gateway, status, version")
		os.Exit(0)
	}

	switch os.Args[1] {
	case "gateway":
		runGateway()
	case "status":
		fmt.Println("Status: not connected (standalone mode)")
	case "version":
		fmt.Println("openclaw-go v0.0.1")
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}
}

func runGateway() {
	cfgPath := config.DefaultConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		log.Printf("Warning: no config loaded (%v), using defaults", err)
		cfg = &config.Config{
			Gateway: config.GatewayConfig{
				Port: 18790, // different from Node OpenClaw to avoid conflict
				Bind: "127.0.0.1",
			},
		}
	}

	addr := fmt.Sprintf("%s:%d", cfg.Gateway.Bind, cfg.Gateway.Port)
	token := os.Getenv("OPENCLAW_TOKEN")

	srv := gateway.NewServer(addr, token)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Graceful shutdown on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("[gateway] shutting down...")
		cancel()
	}()

	log.Printf("[gateway] starting on %s", addr)
	if len(cfg.Agents.List) > 0 {
		log.Printf("[gateway] %d agents configured", len(cfg.Agents.List))
		for _, a := range cfg.Agents.List {
			log.Printf("[gateway]   - %s (%s)", a.Identity.Name, a.ID)
		}
	}

	if err := srv.Start(ctx); err != nil {
		log.Fatalf("[gateway] fatal: %v", err)
	}
}
