#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    -- Create users
    CREATE USER subscriptions_user WITH PASSWORD 'subscriptions_pass';
    CREATE USER news_user WITH PASSWORD 'news_pass';
    CREATE USER accounts_user WITH PASSWORD 'accounts_pass';

    -- Create databases
    CREATE DATABASE subscriptions_db OWNER subscriptions_user;
    CREATE DATABASE news_db OWNER news_user;
    CREATE DATABASE accounts_db OWNER accounts_user;

    -- Grant privileges
    GRANT ALL PRIVILEGES ON DATABASE subscriptions_db TO subscriptions_user;
    GRANT ALL PRIVILEGES ON DATABASE news_db TO news_user;
    GRANT ALL PRIVILEGES ON DATABASE accounts_db TO accounts_user;

    -- Schema privileges
    \c subscriptions_db
    GRANT ALL ON SCHEMA public TO subscriptions_user;

    \c news_db
    GRANT ALL ON SCHEMA public TO news_user;

    \c accounts_db
    GRANT ALL ON SCHEMA public TO accounts_user;
EOSQL

echo "Databases and users created successfully!"
