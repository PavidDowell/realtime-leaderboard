package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	myapi "leaderboard/internal/api"
	"leaderboard/internal/db"
)

func main() {
	ctx := context.Background()

	// connect to Postgres
	pg, err := db.New(ctx)
	if err != nil {
		log.Fatalf("db connect failed: %v", err)
	}
	defer pg.Close()
	log.Println("connected to postgres")

	// connect to Redis
	rd, err := db.NewRedis(ctx)
	if err != nil {
		log.Fatalf("redis connect failed: %v", err)
	}
	defer rd.Close()
	log.Println("connected to redis")

	// create HTTP server
	addr := ":" + getEnv("PORT", "8080")
	srv := myapi.NewServer(addr, pg, rd)

	//run server in a goroutine so we can capture Ctrl+C
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)

	select {
	case sig := <-sigCh:
		log.Printf("received signal %s, shutting down...", sig)
	case err := <-errCh:
		log.Printf("server error: %v", err)
	}

	// this is how you do a graceful shutdown with go
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful shutdown error: %v", err)
	}

	log.Println("api shutdown gracefully")
}

func getEnv(value, def string) string {
	if v := os.Getenv(value); v != "" {
		return v
	}
	return def
}
