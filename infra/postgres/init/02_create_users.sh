#!/bin/bash
set -euo pipefail

create_user() {
  local username="$1"
  local password="$2"

  psql \
    -v ON_ERROR_STOP=1 \
    --username "$POSTGRES_USER" \
    --dbname postgres \
    --set app_user="$username" \
    --set app_password="$password" \
    <<-'EOSQL'
    DO $$
    BEGIN
      IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = :'app_user') THEN
        EXECUTE format('CREATE USER %I WITH PASSWORD %L', :'app_user', :'app_password');
      END IF;
    END
    $$;
EOSQL
}

grant_database() {
  local database="$1"
  local username="$2"

  psql \
    -v ON_ERROR_STOP=1 \
    --username "$POSTGRES_USER" \
    --dbname postgres \
    --set app_db="$database" \
    --set app_user="$username" \
    <<-'EOSQL'
    GRANT ALL PRIVILEGES ON DATABASE :"app_db" TO :"app_user";
EOSQL

  psql \
    -v ON_ERROR_STOP=1 \
    --username "$POSTGRES_USER" \
    --dbname "$database" \
    --set app_user="$username" \
    <<-'EOSQL'
    GRANT ALL ON SCHEMA public TO :"app_user";
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO :"app_user";
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO :"app_user";
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO :"app_user";
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO :"app_user";
EOSQL
}

services=(
  "auth_db ${POSTGRES_AUTH_USER:-auth} ${POSTGRES_AUTH_PASSWORD:-auth}"
  "user_db ${POSTGRES_USER_USER:-user} ${POSTGRES_USER_PASSWORD:-user}"
  "room_db ${POSTGRES_ROOM_USER:-room} ${POSTGRES_ROOM_PASSWORD:-room}"
  "message_db ${POSTGRES_MESSAGE_USER:-message} ${POSTGRES_MESSAGE_PASSWORD:-message}"
)

for service in "${services[@]}"; do
  # shellcheck disable=SC2086
  set -- $service
  create_user "$2" "$3"
  grant_database "$1" "$2"
done
