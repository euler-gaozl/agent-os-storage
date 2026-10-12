package storage

import (
	"bytes"
	"context"
	"github.com/euler-gaozl/agent-os-sdk/contracts"
	"github.com/euler-gaozl/agent-os-sdk/servicehttp"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestArtifactReceiptRetentionAndPageWithoutExecutionTables(t *testing.T) {
	db, role := fixtureDatabase(t)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if err := Migrate(ctx, db, role); err != nil {
		t.Fatal(err)
	}
	cfg := db.Config().Copy()
	cfg.AfterConnect = func(ctx context.Context, c *pgx.Conn) error {
		_, err := c.Exec(ctx, "SET ROLE "+pgx.Identifier{role}.Sanitize())
		return err
	}
	app, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer app.Close()
	store := &Store{DB: app}
	token := strings.Repeat("r", 40)
	handler := (&Server{Store: store, HTTP: servicehttp.Config{Role: "storage", Authorize: Authorize, Peers: map[string]string{"runtime": servicehttp.TokenHash(token)}}}).Handler()
	body := []byte("原文证据与恢复引用")
	hash := contracts.Hash(body)
	root := "run:" + strings.Repeat("a", 32)
	for i := 0; i < 2; i++ {
		request := httptest.NewRequest(http.MethodPut, "/v1/artifacts/"+hash, bytes.NewReader(body))
		request.Header.Set("Authorization", "Bearer "+token)
		request.Header.Set("X-Artifact-Root", root)
		request.Header.Set("X-Artifact-Media-Type", "text/plain")
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != 200 {
			t.Fatalf("save/retry status %d", response.Code)
		}
	}
	var artifacts, refs int
	if err = db.QueryRow(ctx, "SELECT (SELECT count(*) FROM artifacts),(SELECT count(*) FROM artifact_refs)").Scan(&artifacts, &refs); err != nil || artifacts != 1 || refs != 1 {
		t.Fatal("retry duplicated effect", artifacts, refs, err)
	}
	child := "run:" + strings.Repeat("b", 32)
	input := ArtifactPageRequest{Hash: hash, Roots: []string{root, child}, AdoptRoot: child, Offset: 2, Limit: 2}
	page, err := store.ArtifactPage(ctx, input)
	if err != nil || page["content"] != "证据" {
		t.Fatal("unicode page failed", page, err)
	}
	if _, err = store.ArtifactPage(ctx, input); err != nil {
		t.Fatal("page retry", err)
	}
	if err = db.QueryRow(ctx, "SELECT count(*) FROM artifact_refs WHERE root=$1", child).Scan(&refs); err != nil || refs != 1 {
		t.Fatal("adoption duplicated", refs, err)
	}
	other := "run:" + strings.Repeat("c", 32)
	if _, err = store.ArtifactPage(ctx, ArtifactPageRequest{Hash: hash, Roots: []string{other}, AdoptRoot: other}); err == nil {
		t.Fatal("unreferenced root read evidence")
	}
	if _, err = db.Exec(ctx, "UPDATE artifacts SET created_at=now()-interval '48 hours'"); err != nil {
		t.Fatal(err)
	}
	if n, err := store.GC(ctx); err != nil || n != 0 {
		t.Fatal("referenced evidence deleted", n, err)
	}
	saved, _, err := store.Artifact(ctx, hash)
	if err != nil || !bytes.Equal(saved, body) {
		t.Fatal("retained source changed", err)
	}
	if _, err = db.Exec(ctx, "UPDATE artifacts SET body='changed' WHERE hash=$1", hash); err != nil {
		t.Fatal(err)
	}
	if _, _, err = store.Artifact(ctx, hash); err == nil {
		t.Fatal("corrupt source accepted")
	}
}
