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
}

func NewServer(addr string, database *db.Postgres) *Server {
	mux := http.NewServeMux()

	// Attach routes to server
	h := &Handlers{DB: database}
	mux.HandleFunc("GET /healthz", h.healthz)
	mux.HandleFunc("GET /players", h.ListPlayers)
	mux.HandleFunc("POST /score", h.postScore)

	s := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	return &Server{
		httpServer: s,
		DB:         database,
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
