# 历史实现与验收记录

以下为 2026-10-14 整理中文首页前的原始英文记录，保留具体参数、证据编号及历史状态。相对路径以仓库根目录为基准；当前状态以根目录 README 和集成计划为准。

---

# Agent OS Storage

Independent immutable artifact and retention-reference service. Owns only
`public.artifacts` and `public.artifact_refs`. Agent notes/documents, execution
tables and their transactions belong to Runtime; none of that SQL is included
in this repository or the shared SDK.

Runtime/Worker submit content-addressed bytes (up to 4 MiB) with a run retention
root. Artifact and reference commit atomically; a retry adds no duplicate effect.
Runtime supplies authorized roots for Unicode pagination and atomic reference
adoption. Reads verify the stored hash. Service identities restrict mutation,
paging and administrative downloads; these are trusted internal APIs, not an
end-user/tenant authorization system.

The explicit admin GC endpoint only deletes artifacts older than 24 hours with
no retention references. There is no periodic GC or automatic reference removal.
Referenced originals, active recovery evidence and receipts are not deleted by
this service. Expiry of a model context does not revoke retention references.

## Build, migrate and run

Go 1.27.2 / PostgreSQL 18. `scripts/Build.ps1` builds service/migrate/healthcheck
from committed vendor with network module retrieval and go.work disabled.
Linux: `go build -mod=vendor ./cmd/storage`. No sibling source or private token.
The fixed SDK provenance is recorded in `source-provenance.json`.

Supply a non-owner application role and a protected config following
`deploy/service.example.json`. Only two tables need DML rights and public schema
USAGE; the service must not have DDL privileges. Run `bin/migrate --role <role>`
using a separate `MIGRATION_DATABASE_URL`. DDL/grants are a bounded transaction;
the migration does not import documents, rewrite service config or touch other
domains. Normal `bin/storage --config <service.json>` does not migrate.

HTTP defaults to 4 concurrent requests, DB pool to 3. Bind privately, use distinct
tokens/peer hashes (including self), TLS/private-network controls and container
resource limits. `/readyz` checks this schema; `/healthz` only checks liveness.
No Runtime or Registry endpoint is required for this storage process to start.

Retain its DB/config and old binary for independent compatible release/rollback.
`deploy/release-contract.json` identifies compatibility; no destructive down
migrations. Windows integration owns activation and selects a fixed owner commit.

## Verification

Run `go test -mod=vendor -count=1 ./...` and `go vet -mod=vendor ./...`.
With `TEST_DATABASE_URL`, isolated PostgreSQL fixtures verify repeated/failed
migration, least privilege, authenticated save/retry, Unicode pages, reference
adoption, invisible-root denial, retention and corruption checks without any
execution tables. DB tests explicitly skip without PostgreSQL. CI adds Linux
race/build/Docker and Windows build checks. Live release and whole-platform
recovery remain separately reported; a storage test is not an Agent recovery test.

Legacy local configurations may omit the Runtime address: this service preserves
the previous loopback default (127.0.0.1:8094) for its bounded heartbeat/client.
Where used, Registry (8096) and ToolHub (8091) defaults are also owned by the
service entrypoint. Explicit environment/config endpoints always take precedence.
Configure these addresses explicitly for containers or multiple hosts. No config
file or shared SDK role policy is rewritten by this compatibility behavior.
