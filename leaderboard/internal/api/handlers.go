package api

import (
	"context"
	"encoding/json"
	"leaderboard/internal/db"
	"net/http"

	"github.com/redis/go-redis/v9"
)

type Handlers struct {
	DB    *db.Postgres
	Redis *db.Redis
	Hub   *Hub
}

// Health check -> 200 so you know service is alive
func (handler *Handlers) healthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("OK"))
}

// GET /players -> return all players + score from postgres
func (handler *Handlers) ListPlayers(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	rows, err := handler.DB.Pool.Query(ctx, `
		SELECT p.username, COALESCE(ps.score, 0) AS score
		FROM players p
		LEFT JOIN player_scores ps ON ps.player_id = p.id
		ORDER BY score DESC, p.username ASC
		LIMIT 50
	`)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	defer rows.Close()

	type playerRow struct {
		Username string `json:"username"`
		Score    int64  `json:"score"`
	}
	out := []playerRow{}

	for rows.Next() {
		var pr playerRow
		if err := rows.Scan(&pr.Username, &pr.Score); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		out = append(out, pr)
	}
	if rows.Err() != nil {
		http.Error(w, rows.Err().Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(ctx, w, out)
}

func writeJSON(_ context.Context, w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// GET /players -> return all players + score from postgres
func (handler *Handlers) postScore(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	if r.Method != http.MethodPost {
		http.Error(w, "POST Only", http.StatusMethodNotAllowed)
		return
	}

	type scoreReq struct {
		Username       string `json:"username"`
		Delta          int64  `json:"delta"`
		Source         string `json:"source"`
		IdempotencyKey string `json:"idempotency"`
	}

	var req scoreReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if req.Username == "" || req.Delta == 0 || req.Source == "" {
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	// Begin a transaction so that all DB operations are consistent
	tx, err := handler.DB.Pool.Begin(ctx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback(ctx)

	// Ensure the player exists (upsert by username)
	var playerID string
	err = tx.QueryRow(ctx, `
		INSERT INTO players(username)
		VALUES ($1)
		ON CONFLICT(username) DO UPDATE SET username = EXCLUDED.username
		RETURNING id
	`, req.Username).Scan(&playerID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Insert a score event (idempotent via idempotency_key)
	// If the same idempotencyKey is sent twice, ON CONFLICT DO NOTHING
	_, err = tx.Exec(ctx, `
		INSERT INTO score_events(player_id, delta, source, idempotency_key)
		VALUES ($1, $2, $3, NULLIF($4, ''))
		ON CONFLICT(idempotency_key) DO NOTHING
	`, playerID, req.Delta, req.Source, req.IdempotencyKey)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit transaction so trigger runs and player_scores is updated
	if err := tx.Commit(ctx); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Fetch the player's latest total score from player_scores
	var total int64
	err = handler.DB.Pool.QueryRow(ctx, `
		SELECT score
		FROM player_scores
		WHERE player_id = $1
	`, playerID).Scan(&total)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if handler.Redis != nil && handler.Redis.Client != nil {
		redisErr := handler.Redis.Client.ZAdd(ctx, "leaderboard:global",
			redis.Z{
				Score:  float64(total),
				Member: req.Username,
			},
		).Err()
		if redisErr != nil {
			// log it if you want
		}
	}

	if handler.Hub != nil {
		handler.Hub.Broadcast(LeaderboardUpdate{
			Username: req.Username,
			Score:    total,
		})
	}

	type scoreResp struct {
		Username string `json:"username"`
		Score    int64  `json:"score"`
	}
	writeJSON(ctx, w, scoreResp{
		Username: req.Username,
		Score:    total,
	})
}

// GET /leaderboard?limit=10
func (handler *Handlers) getLeaderboard(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	limit := 10
	if handler.Redis == nil || handler.Redis.Client == nil {
		http.Error(w, "redis not available", http.StatusInternalServerError)
		return
	}

	results, err := handler.Redis.Client.ZRevRangeWithScores(ctx, "leaderboard:global", 0, int64(limit-1)).Result()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	type row struct {
		Rank     int    `json:"rank"`
		Username string `json:"username"`
		Score    int64  `json:"score"`
	}
	out := make([]row, 0, len(results))

	for i, z := range results {
		username, _ := z.Member.(string)
		out = append(out, row{
			Rank:     i + 1,
			Username: username,
			Score:    int64(z.Score),
		})
	}

	writeJSON(ctx, w, out)
}
