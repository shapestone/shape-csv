# Contributing to shape-csv

Thank you for your interest in contributing to shape-csv.

## Reporting Bugs

Open an issue on [GitHub Issues](https://github.com/shapestone/shape-csv/issues). Include:

- Go version (`go version`)
- A minimal reproducing example
- Expected vs actual behavior

## Submitting a Pull Request

1. Fork the repository
2. Create a feature branch: `git checkout -b fix/your-description`
3. Make your changes with tests
4. Ensure all checks pass (see below)
5. Open a pull request against `main`

## Development Workflow

Key make targets:

| Target | Description |
|--------|-------------|
| `make test` | Run all tests with race detection — must pass before submitting |
| `make lint` | Run golangci-lint — fix all lint errors before submitting |
| `make bench` | Run benchmarks with memory stats |
| `make coverage` | Generate HTML coverage report |
| `make all` | Run the full check suite (grammar, test, lint, build, coverage) |

Run the full suite before opening a PR:

```bash
make all
```

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- All exported functions and types must have doc comments
- New features must include tests; bug fixes must include a regression test
- Keep allocations minimal — run `make bench` before and after performance-sensitive changes

## Questions

Open a [GitHub Issue](https://github.com/shapestone/shape-csv/issues) and tag it with the `question` label.
