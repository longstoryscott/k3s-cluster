package storage

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

var DB *pgxpool.Pool

func InitDB(connString string) error {
	var err error

	// Create a connection pool config with reasonable defaults
	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("unable to parse database config: %w", err)
	}

	// Set reasonable pool size
	config.MaxConns = 10

	// Create the connection pool
	DB, err = pgxpool.NewWithConfig(context.Background(), config)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %w", err)
	}

	// Verify connection works
	if err := DB.Ping(context.Background()); err != nil {
		return fmt.Errorf("unable to ping database: %w", err)
	}

	log.Println("Successfully connected to database with connection pool")
	return nil
}

// InitSchema creates the necessary tables if they don't exist
func InitSchema(ctx context.Context) {
	// Create users table
	_, err := DB.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS users (
            id TEXT PRIMARY KEY,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )
    `)
	if err != nil {
		panic(err)
	}

	log.Println("Creating users table")

	// Create conversations table
	_, err = DB.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS conversations (
            id SERIAL PRIMARY KEY,
            user_id TEXT NOT NULL REFERENCES users(id),
            title TEXT,
            model TEXT NOT NULL,
            created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
            updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
        )
    `)
	if err != nil {
		panic(err)
	}

	log.Println("Creating conversations table")

	// Create messages table
	_, err = DB.Exec(ctx, `
        CREATE TABLE IF NOT EXISTS messages (
            conversation_id INTEGER NOT NULL REFERENCES conversations(id),
            role TEXT NOT NULL,  -- 'user' or 'assistant'
            content TEXT NOT NULL,
            created_at TIMESTAMPTZ PRIMARY KEY DEFAULT NOW()
        )
    `)
	if err != nil {
		panic(err)
	}

	log.Println("Creating messages table")

	// Convert messages to a TimescaleDB hypertable
	_, err = DB.Exec(ctx, `
        SELECT create_hypertable('messages', 'created_at', if_not_exists => TRUE)
    `)
	if err != nil {
		panic(err)
	}

	log.Println("Creating messages hypertable")
}
