package db

import (
	"context"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Postgres struct {
	Pool *pgxpool.Pool
}

func New(ctx context.Context) (*Postgres, error) {
	dsn := os.Getenv("PG_URI")
	if dsn == "" {
		dsn = "postgres://postgres:postgres@localhost:5432/postgres?sslmode=disable"
	}

	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	var pool *pgxpool.Pool
	// retry if unable to connect
	for i := 0; i < 10; i++ {
		pool, err = pgxpool.NewWithConfig(ctx, cfg)
		if err == nil {
			pingErr := pool.Ping(ctx)
			if pingErr := pool.Ping(ctx); pingErr == nil {
				// success
				return &Postgres{Pool: pool}, nil
			}
			// ping failed, close and treat like error
			pool.Close()
			err = pingErr
		}

		// wait a bit before retry
		time.Sleep(500 * time.Millisecond)
	}

	// after 10 tries, give up and return last error
	return nil, err
}

func (p *Postgres) Close() {
	p.Pool.Close()
}
