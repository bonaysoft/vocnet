# Repository Guidelines

## Project Structure & Module Organization
- `cmd/` hosts Cobra entry points (`serve`, `db-init`); `main.go` delegates to `cmd.Execute()`.
- `api/proto` defines service contracts; regenerate outputs into `api/gen` and `api/openapi` before committing.
- `internal/` follows Clean Architecture: `entity/` for core models, `usecase/` for orchestration, `adapter/` (`grpc/`, `repository/`) for delivery & persistence, `infrastructure/` (`config/`, `database/`, `server/`) for frameworks, with `mocks/` supplying GoMock doubles.
- `internal/infrastructure/database/entschema/` holds ent schema definitions; generated code resides in `internal/infrastructure/database/ent/`.

## Build, Test, and Development Commands
- `make setup` installs buf/mockgen, downloads modules, and refreshes generated assets.
- `make run` launches the gRPC + HTTP gateway in place; `make build` outputs `bin/vocnet` for distribution.
- `make db-up`/`make migrate` provision Docker PostgreSQL and apply ent migrations; `make dev` sequences DB bootstrap then server startup.
- `make test` runs `go test -race -cover ./...`; `make lint` executes `golangci-lint run`. Regenerate protobuf和 ent 代码通过 `make generate`。

## Coding Style & Architecture Conventions
- Keep Go files `gofmt`/`goimports` clean (`make fmt`); use tabs and grouped imports.
- Preserve Clean Architecture boundaries—the inner layers never import adapters or infrastructure; prefer dependency injection for wiring.
- Use `context.Context` on all IO paths, wrap errors with `%w`, and document exported packages; keep RPC names imperative (`CollectWord`).
- Log via `logrus.WithField` and avoid hardcoded configuration, relying on Viper-loaded `.env` values.

## Testing Guidelines
- Co-locate tests (`*_test.go`) and favour table-driven subtests; mock outbound calls with GoMock from `internal/mocks/`.
- Keep coverage trending upward (≈80%+ for new code) and extend integration checks via CLI smoke tests in `cmd/db_init_test.go` when touching database flows.
- Exercise both success and failure paths; always run `make test` before submitting and commit the resulting `coverage.out` when behavior changes.

## Commit & Pull Request Guidelines
- Follow Conventional Commits (`feat:`, `fix:`, `docs:`) with optional scopes (`feat(adapter): ...`).
- PRs must describe behavior changes, reference issues, flag schema/API updates, and include the latest `make test` output (screenshots only for doc/UI changes).
- Regenerate and commit protobuf、ent 生成代码或 mocks alongside source updates.

## Tooling & Verification Notes
- Validate protobuf compatibility with `make buf-lint` and `make buf-breaking` before publishing interface changes.
- ent schemas live in `internal/infrastructure/database/entschema/`; run `make ent-generate` when schema files change.
- New configuration keys should be documented in `docs/` and surface through Viper rather than hardcoded constants.
