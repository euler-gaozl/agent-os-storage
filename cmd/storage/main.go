package main

import (
	"context"
	"github.com/euler-gaozl/agent-os-sdk/servicehost"
	"github.com/euler-gaozl/agent-os-sdk/servicehttp"
	"github.com/euler-gaozl/agent-os-storage/storage"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
)

func main() {
	servicehost.Main(servicehost.Options{Role: "storage", Port: "8092", Database: true, Concurrency: 4, Authorize: storage.Authorize, Readiness: storage.Readiness}, func(ctx context.Context, p *pgxpool.Pool, c servicehttp.Config) (http.Handler, func(context.Context) error, error) {
		return (&storage.Server{Store: &storage.Store{DB: p}, HTTP: c}).Handler(), nil, nil
	})
}
