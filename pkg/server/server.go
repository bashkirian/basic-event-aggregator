// pkg/server/server.go
package server

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/bashkirian/event-aggregator/internal/aggregator"
	"github.com/bashkirian/event-aggregator/internal/handler"
	"github.com/bashkirian/event-aggregator/internal/storage"
)

type Server struct {
	httpServer *http.Server
	aggregator *aggregator.Aggregator
}

func NewServer(port string) *Server {
	return NewServerWithStorage(port, nil)
}

func NewServerWithStorage(port string, redisAddr string) *Server {
	return NewServerWithRedisConfig(port, redisAddr, "", 0)
}

func NewServerWithRedisConfig(port, redisAddr, redisPassword string, redisDB int) *Server {
	var store storage.Storage
	
	if redisAddr != "" {
		// Используем Redis хранилище
		store = storage.NewRedisStorage(redisAddr, redisPassword, redisDB)
	} else {
		// Используем in-memory хранилище по умолчанию
		store = storage.NewInMemoryStorage()
	}
	
	agg := aggregator.New(store, 1000)
	h := handler.New(agg)

	mux := http.NewServeMux()
	mux.HandleFunc("/events", h.HandlePostEvent)
	mux.HandleFunc("/aggregated", h.HandleGetAggregated)
	mux.HandleFunc("/aggregated/all", h.HandleGetAllAggregated)
	mux.HandleFunc("/health", h.HandleHealth)

	httpServer := &http.Server{
		Addr:         ":" + port,
		Handler:      loggingMiddleware(mux),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
		aggregator: agg,
	}
}

func (s *Server) Start() error {
	s.aggregator.Start(context.Background())
	log.Printf("Server starting on %s", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("%s %s", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
		log.Printf("%s %s - completed in %v", r.Method, r.URL.Path, time.Since(start))
	})
}
