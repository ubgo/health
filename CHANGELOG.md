# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Initial implementation of the `health` core module.
- `Status` (`Up`, `Down`, `Degraded`, `Unknown`), `Severity` (`Critical`, `Degraded`, `Info`), `Probe` (`Liveness`, `Readiness`, `Startup`) types.
- `Result` and `Checker` interface as the integration point for dependency checks.
- Thread-safe `Registry` with `NewRegistry`, `Register` (with `WithSeverity` / `WithTimeout` options), `Unregister`, `Run`, `RunAll`, `Snapshot`, `Subscribe`.
- Background re-check loop via `Start(ctx, interval)` / `Stop` — idempotent, safe under concurrent restarts.
- `Snapshot` aggregate with probe-aware status reduction (`SnapshotForProbe`), `HTTPStatus` (200/503), and `JSON` marshalling.
- Observer pattern for adapters that need to react after every check (OTEL spans, Prometheus metrics, alerting webhooks).
- 90.7% statement coverage with race detector enforced on every test.
- Taskfile, CI workflows, README, NOTICE.
- Licensed under Apache License 2.0.

[Unreleased]: https://github.com/ubgo/health/compare/v0.0.0...HEAD
