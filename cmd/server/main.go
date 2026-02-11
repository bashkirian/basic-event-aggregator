// cmd/server/main.go
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/bashkirian/event-aggregator/internal/config"
	"github.com/bashkirian/event-aggregator/pkg/server"
)

func main() {
	// Загружаем конфигурацию
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	var srv *server.Server
	if cfg.Redis.Addr != "" {
		log.Printf("Using Redis storage at %s", cfg.Redis.Addr)
		srv = server.NewServerWithRedisConfig(cfg.Server.Port, cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
	} else {
		log.Println("Using in-memory storage")
		srv = server.NewServer(cfg.Server.Port)
	}

	// Start server
	go func() {
		if err := srv.Start(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server failed to start: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	
	log.Println("Server shutdown complete")
}
