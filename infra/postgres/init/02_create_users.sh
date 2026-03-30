#!/bin/bash
set -euo pipefail

create_user() {
  local username="$1"
  local password="$2"
  local quoted_username="\"$username\""

  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres -c "
    DO \$\$
    BEGIN
      IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = '$quoted_username') THEN
        CREATE USER $quoted_username WITH PASSWORD '$password';
      END IF;
    END
    \$\$;"
}

grant_database() {
  local database="$1"
  local username="$2"
  local quoted_username="\"$username\""

  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres -c "GRANT ALL PRIVILEGES ON DATABASE $database TO $quoted_username;"

  psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$database" -c "
    GRANT ALL ON SCHEMA public TO $quoted_username;
    GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO $quoted_username;
    GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO $quoted_username;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO $quoted_username;
    ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO $quoted_username;"
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
