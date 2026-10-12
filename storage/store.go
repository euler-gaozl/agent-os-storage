package storage

import (
	"context"
	"errors"
	. "github.com/euler-gaozl/agent-os-sdk/contracts"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func PutArtifact(ctx context.Context, tx pgx.Tx, body []byte, media, root string) (string, error) {
	if len(body) > 4<<20 || root == "" {
		return "", errors.New("artifact requires retention root and maximum 4 MiB")
	}
	hash := Hash(body)
	_, e := tx.Exec(ctx, "INSERT INTO artifacts(hash,body,media_type) VALUES($1,$2,$3) ON CONFLICT DO NOTHING", hash, body, media)
	if e != nil {
		return "", e
	}
	_, e = tx.Exec(ctx, "INSERT INTO artifact_refs(root,hash) VALUES($1,$2) ON CONFLICT DO NOTHING", root, hash)
	return hash, e
}
func (s *Store) SaveArtifact(ctx context.Context, body []byte, media, root string) (string, error) {
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return "", e
	}
	defer tx.Rollback(ctx)
	hash, e := PutArtifact(ctx, tx, body, media, root)
	if e != nil {
		return "", e
	}
	return hash, tx.Commit(ctx)
}
func (s *Store) Artifact(ctx context.Context, hash string) ([]byte, string, error) {
	var body []byte
	var media string
	e := s.DB.QueryRow(ctx, "SELECT body,media_type FROM artifacts WHERE hash=$1", hash).Scan(&body, &media)
	if e == nil && Hash(body) != hash {
		e = errors.New("artifact integrity check failed")
	}
	return body, media, e
}
func (s *Store) GC(ctx context.Context) (int64, error) {
	tag, e := s.DB.Exec(ctx, "DELETE FROM artifacts a WHERE a.created_at<now()-interval '24 hours' AND NOT EXISTS(SELECT 1 FROM artifact_refs r WHERE r.hash=a.hash)")
	return tag.RowsAffected(), e
}

type Store struct{ DB *pgxpool.Pool }
