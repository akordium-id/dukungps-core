#!/bin/sh
set -e

# wait for postgres, then apply migrations if schema is not complete
if [ -n "$DATABASE_URL" ]; then
  echo "applying migrations..."

  # parse DATABASE_URL into pg env vars (keeps password out of psql argv)
  # postgres://user:pass@host:port/db?params
  HOST=$(echo "$DATABASE_URL" | sed -n 's|postgres://[^@]*@\([^:/?]*\)[:/?].*|\1|p')
  PORT=$(echo "$DATABASE_URL" | sed -n 's|postgres://[^@]*@[^:]*:\([0-9]*\)[/?].*|\1|p')
  DB=$(echo "$DATABASE_URL" | sed -n 's|postgres://[^@]*@[^/]*/\([^?]*\).*|\1|p')
  USER=$(echo "$DATABASE_URL" | sed -n 's|postgres://\([^:]*\):.*|\1|p')
  PASSWORD=$(echo "$DATABASE_URL" | sed -n 's|postgres://[^:]*:\([^@]*\)@.*|\1|p')
  : "${PORT:=5432}"

  export PGHOST="$HOST"
  export PGPORT="$PORT"
  export PGDATABASE="$DB"
  export PGUSER="$USER"
  export PGPASSWORD="$PASSWORD"

  i=0
  until pg_isready -q; do
    i=$((i+1))
    if [ "$i" -ge 30 ]; then
      echo "postgres not ready after 30s"
      exit 1
    fi
    sleep 1
  done

  # completeness gate: only apply when the schema is missing entirely.
  # Runs in a single transaction so partial state cannot be skipped forever.
  if [ -n "$DB" ] && ! psql -tAc "SELECT to_regclass('public.devices') IS NOT NULL AND to_regclass('public.positions') IS NOT NULL AND to_regclass('public.users') IS NOT NULL" | grep -q "t"; then
    psql --single-transaction --set ON_ERROR_STOP=1 -f /migrations/001_initial_schema.sql
    echo "migrations applied"
  else
    echo "migrations skipped (already applied)"
  fi
fi

exec ./server
