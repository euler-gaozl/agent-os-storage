package storage

import (
	"context"
	"errors"
	"regexp"

	. "github.com/euler-gaozl/agent-os-sdk/contracts"
	"github.com/euler-gaozl/agent-os-sdk/servicehttp"
	"github.com/jackc/pgx/v5"
)

var RunRoot = regexp.MustCompile(`^run:[a-f0-9]{32}$`)
var ArtifactHash = regexp.MustCompile(`^[a-f0-9]{64}$`)

// A read-domain rejection is distinct from HTTP routing/auth/transport failures.
// Only these known checks may become replayable model feedback.
type artifactReadRejection struct{ Code, Message string }

func (e *artifactReadRejection) Error() string { return e.Message }

// Roots are computed by Runtime from its execution hierarchy, never accepted
// from a model or browser. Storage owns references but does not query runs.
type ArtifactPageRequest struct {
	Hash      string   `json:"hash"`
	Offset    int      `json:"offset"`
	Limit     int      `json:"limit"`
	Roots     []string `json:"roots"`
	AdoptRoot string   `json:"adopt_root"`
}

func (s *Store) ArtifactPage(ctx context.Context, in ArtifactPageRequest) (map[string]any, error) {
	if !ArtifactHash.MatchString(in.Hash) || !RunRoot.MatchString(in.AdoptRoot) || len(in.Roots) == 0 || len(in.Roots) > 128 || in.Offset < 0 || in.Limit < 0 || in.Limit > 1200 {
		return nil, errors.New("bounded artifact page and run roots required")
	}
	owned := false
	for _, root := range in.Roots {
		if !RunRoot.MatchString(root) {
			return nil, errors.New("invalid artifact visibility root")
		}
		owned = owned || root == in.AdoptRoot
	}
	if !owned {
		return nil, errors.New("adopt root must be within visible roots")
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return nil, e
	}
	defer tx.Rollback(ctx)
	var body []byte
	var media string
	e = tx.QueryRow(ctx, `SELECT a.body,a.media_type FROM artifacts a WHERE a.hash=$1 AND EXISTS(
SELECT 1 FROM artifact_refs r WHERE r.hash=a.hash AND r.root=ANY($2)) FOR KEY SHARE OF a`, in.Hash, in.Roots).Scan(&body, &media)
	if e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			// Missing and outside the visible roots deliberately look identical.
			return nil, &artifactReadRejection{Code: "artifact_unavailable", Message: "No readable runtime artifact for this hash. raw_sha256 and original_hash identify source bytes, not artifact_read handles. Use document_read with the knowledge document id, or the bound Sources read_capture tool with its capture id."}
		}
		return nil, e
	}
	if Hash(body) != in.Hash {
		return nil, &servicehttp.StatusError{Status: 500, Message: "artifact integrity check failed"}
	}
	chars := []rune(string(body))
	if in.Offset > len(chars) {
		return nil, &artifactReadRejection{Code: "artifact_window_invalid", Message: "The offset is outside this runtime artifact. Read offset 0 first and follow next_offset; capture and document offsets are not artifact offsets."}
	}
	if in.Limit == 0 {
		in.Limit = 1200
	}
	end := min(in.Offset+in.Limit, len(chars))
	if _, e = tx.Exec(ctx, "INSERT INTO artifact_refs(root,hash) VALUES($1,$2) ON CONFLICT DO NOTHING", in.AdoptRoot, in.Hash); e != nil {
		return nil, e
	}
	result := map[string]any{"hash": in.Hash, "media_type": media, "content": string(chars[in.Offset:end]), "total_characters": len(chars), "next_offset": end, "has_more": end < len(chars)}
	return result, tx.Commit(ctx)
}
