# tinytoe -- A Vibe-Coded Database Migration Tool

## AI Disclaimer

This is written almost entirely by Codex CLI, with code reviews and check-ins by me.

## Integration Tests

Run `./scripts/run-integration-tests.sh` to spin up a disposable PostgreSQL container and execute the Go test suite against it. The script binds the database to a random high port (backed by container port 5544) so client code must respect `PGPORT`/`DATABASE_URL`, and it tears the container down automatically on exit.

## Unit Tests

`./scripts/run-unit-tests.sh` drives `go test ./...` with local caches so it stays sandbox-friendly. The script prefers [`gotestsum`](https://github.com/gotesttools/gotestsum) for its richer output; install it with `go install gotest.tools/gotestsum@latest`. If the binary is missing and the module cannot be downloaded, the script falls back to plain `go test`.

## Dependencies

We pin the toolchain in `go.mod` (`toolchain go1.24.3`). Running `go version` should report a compatible release; if not, install it via `go install golang.org/dl/go1.24.3@latest` followed by `go1.24.3 download`. 

To upgrade a dependency, use `go get module@version`, re-run the test scripts, and include the resulting module diffs in your PR. After adding or removing imports, run `go mod tidy` so `go.mod` and `go.sum` stay in sync. 

To sweep everything at once, run:

```bash
go get -u -t ./...
go mod tidy
```

Our helper scripts keep caches inside `.gocache` and `.gomod`, so there is no need for global cache tweaking.

## Interactive Tests

1. Create a local `.env` symlink so the CLI picks up the example configuration.
2. Launch the development database with Docker.
3. Run the CLI through the helper script as needed.

```bash
ln -s env.codex .env
docker compose up -d
./pinkytoe init
```

## License

This software is available under the Zero-Clause BSD (0BSD) license. See `LICENSE` for the full text.
