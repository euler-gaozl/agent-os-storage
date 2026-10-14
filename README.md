# Agent OS Storage

独立的不可变产物与保留引用服务，只拥有 `public.artifacts`、`public.artifact_refs`。Agent 笔记、执行记录及其原子事务属于 Runtime，不在此服务或 SDK 中复制。

## 数据行为

Runtime/Worker 提交最大 4 MiB 的内容寻址字节与运行保留根。产物和引用原子提交，重复提交不产生重复效果；读取核验存储哈希，并支持按授权根进行 Unicode 分页和原子引用继承。内部服务身份限制写入、分页和管理下载，不等于终端用户/租户权限体系。

`POST /v1/artifacts/page` 对实际不存在/不可见的产物统一返回明确的读取结果 `{"isError":true,"code":"artifact_unavailable","message":"…"}`；越界读取为 `artifact_window_invalid`。这两类结果在引用写入前确定，HTTP 200 使 Runtime 能将纠错反馈按原操作落盘并重放。路由、权限、数据库、传输和完整性错误仍是 HTTP 错误，不能把任意 404/416 当作业务拒绝。源码哈希不自动获得本任务的运行产物读取权限。2026-10-14 隔离 PostgreSQL 与 HTTP 验证 5 项通过；现有 Runtime 的纠正、回执复用和拒绝路径另行验证，无须修改执行器。

只有显式管理 GC 接口能删除超过 24 小时且完全没有引用的产物。**不定时自动 GC，也不自动移除引用**。模型上下文被压缩不会撤销原文、回执或恢复证据的保留引用。

## 构建与部署

需要 Go 1.27.2、PostgreSQL 18。

```powershell
$env:GOWORK='off'; $env:GOPROXY='off'
./scripts/Build.ps1
go test -mod=vendor -count=1 ./...
go vet -mod=vendor ./...
```

Linux 可用 `go build -mod=vendor ./cmd/storage`。固定 SDK 和 `vendor` 使构建不依赖相邻源码或私有令牌。按 [配置示例](deploy/service.example.json) 提供非所有者数据库账号、独立服务令牌和对端哈希（含自身），使用 `bin/storage --config <文件>` 启动。

应用账号仅需两张表的 DML 和 public schema USAGE，不能有 DDL 权限。单独通过 `MIGRATION_DATABASE_URL` 执行 `bin/migrate --role <账号>`，在有界事务里完成本域 DDL/授权；不导入文档、不改其他域或服务配置，正常启动不迁移。

HTTP 并发默认 4，连接池默认 3。服务不要求 Runtime/Registry 地址才能启动；兼容配置可保留 Runtime 回环心跳默认值，多机应明确设置地址。内部端口使用私网/TLS 和资源限制。`/readyz` 检查本域，`/healthz` 仅检查存活。

## 升级与验证

保留数据库、配置与旧程序，按 [发布契约](deploy/release-contract.json) 独立升级/回滚，不执行破坏性反向迁移。Windows 集成平台固定服务提交并控制激活。

设置 `TEST_DATABASE_URL` 后，隔离 PostgreSQL 测试覆盖迁移回滚、最小权限、保存重试、Unicode 分页、引用继承、不可见根拒绝、保留与损坏检查；未设置则明确跳过。Storage 测试不等于完整 Agent 恢复验收。Actions 日常关闭，[历史验证细节](docs/history/service-notes-20261014.en.md) 保留原始记录。
