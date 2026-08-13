# Contributing to gaardrail

Thanks for your interest in contributing!

## Development setup

Requirements: Go (version from `go.mod`); Docker only for the optional local Kafka and image builds.

```bash
git clone https://github.com/adrianozp/gaardrail
cd gaardrail
go build ./...
go test ./...
```

Run the app with `make run` (reads `config/config.yaml`). The default config uses `queue.protocol: constant`, which needs no broker. For a real queue: `make kafka/up && make kafka/setup`, then set `queue.protocol: kafka`.

## Linting

CI runs [golangci-lint](https://golangci-lint.run). Locally:

```bash
go run github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest run ./...
```

## Pull requests

1. Fork and branch from `main`.
2. Keep changes focused; add or update tests for behavior changes.
3. Make sure `go build ./...`, `go test ./...` and the linter pass.
4. Open the PR describing the motivation and the change.

## Commit style

Short imperative subject lines ("Add X", "Fix Y"). Reference issues when applicable.

## Bugs and features

Use the issue templates. For security issues see [SECURITY.md](SECURITY.md) — do not open a public issue.

## License

By contributing you agree that your contributions are licensed under the [MPL-2.0](LICENSE).
