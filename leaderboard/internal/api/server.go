package api

import (
	"context"
	"leaderboard/internal/db"
	"log"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
	DB         *db.Postgres
	Redis      *db.Redis
}

func NewServer(addr string, database *db.Postgres, cache *db.Redis) *Server {
	mux := http.NewServeMux()

	// Attach routes to server
	handler := &Handlers{
		DB:    database,
		Redis: cache,
	}
	mux.HandleFunc("GET /healthz", handler.healthz)
	mux.HandleFunc("GET /players", handler.ListPlayers)
	mux.HandleFunc("POST /score", handler.postScore)
	mux.HandleFunc("GET /leaderboard", handler.getLeaderboard)

	s := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{
		httpServer: s,
		DB:         database,
		Redis:      cache,
	}
}

// Start blocks and serves HTTP.
func (s *Server) Start() error {
	log.Printf("listening on %s\n", s.httpServer.Addr)
	return s.httpServer.ListenAndServe()
}

// Shutdown lets you gracefully stop later if you add signals, etc.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.httpServer.Shutdown(ctx)
}
