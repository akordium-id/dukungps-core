#!/bin/sh
set -e

# wait for postgres, then apply migrations if not yet applied
if [ -n "$DATABASE_URL" ]; then
  echo "applying migrations..."
  # extract host/port/db from DATABASE_URL (postgres://user:pass@host:port/db)
  HOST=$(echo "$DATABASE_URL" | sed -n 's|postgres://[^@]*@\([^:/]*\)[:/].*|\1|p')
  PORT=$(echo "$DATABASE_URL" | sed -n 's|postgres://[^@]*@[^:]*:\([0-9]*\)/.*|\1|p')
  DB=$(echo "$DATABASE_URL" | sed -n 's|postgres://[^@]*@[^/]*/\(.*\)|\1|p')
  : "${PORT:=5432}"

  i=0
  until pg_isready -h "$HOST" -p "$PORT" -q; do
    i=$((i+1))
    if [ "$i" -ge 30 ]; then
      echo "postgres not ready after 30s"
      exit 1
    fi
    sleep 1
  done

  if [ -n "$DB" ] && ! psql "$DATABASE_URL" -tAc "SELECT 1 FROM devices LIMIT 1" >/dev/null 2>&1; then
    psql "$DATABASE_URL" -f /migrations/001_initial_schema.sql
    echo "migrations applied"
  else
    echo "migrations skipped (already applied)"
  fi
fi

exec ./server
