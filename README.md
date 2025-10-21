# Tiny Toe Database Migration Tool

## PostgreSQL Setup Script

Run the `scripts/setup_postgres.sh` script once to install and configure PostgreSQL for the Codex environment.

```bash
sudo ./scripts/setup_postgres.sh
```

The script performs a straight-through installation:

- Installs PostgreSQL and contrib packages via `apt-get`.
- Starts the PostgreSQL service.
- Appends password-authentication rules for localhost to `pg_hba.conf`.
- Creates the `tt` role with password `tt` and the `tt` database owned by that role.

After the script completes, the `DATABASE_URL` defined in `env.codex` (`postgresql://tt:tt@localhost/tt`) will be ready for use.
