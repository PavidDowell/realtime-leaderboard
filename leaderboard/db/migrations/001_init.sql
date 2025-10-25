CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE players (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  username   TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE score_events (
  id              BIGSERIAL PRIMARY KEY,
  player_id       UUID NOT NULL REFERENCES players(id),
  delta           BIGINT NOT NULL,
  source          TEXT NOT NULL,
  occurred_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  idempotency_key TEXT UNIQUE
);

CREATE TABLE player_scores (
  player_id  UUID PRIMARY KEY REFERENCES players(id),
  score      BIGINT NOT NULL DEFAULT 0,
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE OR REPLACE FUNCTION apply_score()
RETURNS TRIGGER AS $$
BEGIN
  INSERT INTO player_scores(player_id, score, updated_at)
  VALUES (NEW.player_id, NEW.delta, now())
  ON CONFLICT (player_id) DO UPDATE
  SET score = player_scores.score + EXCLUDED.score,
      updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tg_apply_score
AFTER INSERT ON score_events
FOR EACH ROW EXECUTE FUNCTION apply_score();
