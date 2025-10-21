# tinytoe -- A Vibe-Coded Database Migration Tool

## AI Disclaimer

This is written almost entirely by Codex CLI, with code reviews and check-ins by me.

## Integration Tests

Run `./scripts/run-integration-tests.sh` to spin up a disposable PostgreSQL container and execute the Go test suite against it. The script binds the database to a random high port (backed by container port 5544) so client code must respect `PGPORT`/`DATABASE_URL`, and it tears the container down automatically on exit.

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
