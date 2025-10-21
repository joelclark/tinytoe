#!/usr/bin/env bash
set -euo pipefail

if [ "$(id -u)" -ne 0 ]; then
  echo "Run this script as root (e.g., via sudo)." >&2
  exit 1
fi

apt-get update
apt-get install -y postgresql postgresql-contrib

service postgresql start

PG_VERSION=$(ls /etc/postgresql | sort -nr | head -n1)
PG_HBA="/etc/postgresql/${PG_VERSION}/main/pg_hba.conf"

cat <<'CONF' >> "$PG_HBA"
host all all 127.0.0.1/32 md5
host all all ::1/128 md5
CONF

service postgresql reload

sudo -u postgres psql <<'SQL'
CREATE ROLE tt WITH LOGIN PASSWORD 'tt';
CREATE DATABASE tt OWNER tt;
SQL

echo "PostgreSQL installed and configured."
