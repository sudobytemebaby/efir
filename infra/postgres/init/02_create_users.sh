#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" <<-EOSQL
    CREATE USER auth WITH PASSWORD '${POSTGRES_AUTH_PASSWORD:-auth}';
    CREATE USER user WITH PASSWORD '${POSTGRES_USER_PASSWORD:-user}';
    CREATE USER room WITH PASSWORD '${POSTGRES_ROOM_PASSWORD:-room}';
    CREATE USER message WITH PASSWORD '${POSTGRES_MESSAGE_PASSWORD:-message}';

    GRANT ALL PRIVILEGES ON DATABASE auth_db TO auth;
    GRANT ALL PRIVILEGES ON DATABASE user_db TO user;
    GRANT ALL PRIVILEGES ON DATABASE room_db TO room;
    GRANT ALL PRIVILEGES ON DATABASE message_db TO message;

    CONNECT auth_db;
    GRANT ALL ON SCHEMA public TO auth;

    CONNECT user_db;
    GRANT ALL ON SCHEMA public TO user;

    CONNECT room_db;
    GRANT ALL ON SCHEMA public TO room;

    CONNECT message_db;
    GRANT ALL ON SCHEMA public TO message;
EOSQL
