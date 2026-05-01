# Contributing to ubgo/health

Thanks for your interest in `ubgo/health`. This repository is licensed under the **Apache License 2.0**. Pull requests are welcome.

## Ways to contribute

- **Bug reports.** Open a GitHub issue describing the unexpected behaviour, the Go version, OS / arch, and a minimal reproduction.
- **Feature proposals.** Open a GitHub issue first describing the use case and proposed API. We'll iterate on the design before code lands.
- **Pull requests.** See the workflow below.

## Workflow

1. Open an issue first for anything beyond a tiny fix. Discussing the design upfront avoids wasted work.
2. Fork the repository.
3. Create a branch named after the issue: `fix/123-stop-race`, `feat/456-grpc-adapter`.
4. Run the local checks before opening the PR:

   ```sh
   task ci          # vet + race tests + zero-dep gate
   ```

5. Use Conventional Commits for the PR title:
   - `feat(health): add WithDependencies registration option`
   - `fix(registry): handle nil checker gracefully`
   - `docs(readme): clarify probe semantics`
   - `chore(deps): bump golangci-lint`

## Code conventions

- **Zero third-party deps in the core.** The `zero-dep-gate` CI check fails if `go.mod` of the core module gains any non-stdlib `require` line. Adapter modules under `contrib/` are free to depend on their target ecosystems.
- **Race detector mandatory.** Every test must pass under `-race`.
- **Coverage target.** Core: ≥ 90% line coverage. Adapters: ≥ 80%.
- **Public API stability.** Once the module reaches v1.0.0, breaking changes require a major version bump and a strong rationale in the PR description.
- **No comments explaining what the code does.** Names should make that clear. Reserve comments for the *why* — non-obvious invariants, hidden constraints, surprising tradeoffs.

## Testing locally

```sh
task test           # standard tests
task test:race      # with race detector
task test:coverage  # with coverage report
task lint           # golangci-lint
task ci             # everything
```

## Reporting security issues

Please do **not** open a public GitHub issue for security vulnerabilities. Email the maintainer directly. We'll respond within 7 days and coordinate disclosure.

## License of contributions

By submitting a pull request, you agree that your contribution is provided under the same Apache License 2.0 as the rest of the repository (per the standard "inbound = outbound" rule, codified in section 5 of the Apache License).
