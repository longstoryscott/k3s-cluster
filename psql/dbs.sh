#!/bin/bash
set -e
# Create databases
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "postgres" <<-EOSQL
  CREATE DATABASE IF NOT EXISTS ollama;
  CREATE DATABASE IF NOT EXISTS dex_db;
  
  -- Create TimescaleDB extension in each database
  \c ollama
  CREATE EXTENSION IF NOT EXISTS timescaledb;
  
  \c dex_db  
  CREATE EXTENSION IF NOT EXISTS timescaledb;
EOSQL
