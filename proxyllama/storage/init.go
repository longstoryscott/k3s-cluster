package storage

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	DB       *pgxpool.Pool
	initOnce sync.Once = sync.Once{}
)

// InitDB initializes the database connection
func InitDB(connString string) error {
	var err error
	initOnce.Do(func() {
		// Create a connection pool config with reasonable defaults
		config, err := pgxpool.ParseConfig(connString)
		if err != nil {
			err = fmt.Errorf("unable to parse database config: %w", err)
			return
		}

		// Set reasonable pool size
		config.MaxConns = 10

		// Create the connection pool
		DB, err = pgxpool.NewWithConfig(context.Background(), config)
		if err != nil {
			err = fmt.Errorf("unable to connect to database: %w", err)
			return
		}

		// Verify connection works
		if err := DB.Ping(context.Background()); err != nil {
			err = fmt.Errorf("unable to ping database: %w", err)
			return
		}

		log.Println("Successfully connected to database with connection pool")
	})
	return err
}

// InitSchema creates the database schema if it doesn't exist
func InitSchema(ctx context.Context) {
	// Create users table
	_, err := DB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Printf("Error creating users table: %v", err)
	}

	// Create conversations table
	_, err = DB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS conversations (
			id SERIAL PRIMARY KEY,
			user_id TEXT NOT NULL REFERENCES users(id),
			title TEXT DEFAULT 'New conversation',
			model TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Printf("Error creating conversations table: %v", err)
	}

	// Create messages table
	_, err = DB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS messages (
			id SERIAL PRIMARY KEY,
			conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Printf("Error creating messages table: %v", err)
	}

	// Create summaries table
	_, err = DB.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS summaries (
			id SERIAL PRIMARY KEY,
			conversation_id INTEGER NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
			content TEXT NOT NULL,
			level INTEGER NOT NULL,
			source_ids JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Printf("Error creating summaries table: %v", err)
	}

	// Update conversation deletion function to clean up messages
	_, err = DB.Exec(ctx, `
		-- Update conversation.update_at when a new message is added
		CREATE OR REPLACE FUNCTION update_conversation_updated_at()
		RETURNS TRIGGER AS $$
		BEGIN
			UPDATE conversations
			SET updated_at = NOW()
			WHERE id = NEW.conversation_id;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
	`)
	if err != nil {
		log.Printf("Error creating update_conversation_updated_at function: %v", err)
	}

	// Create trigger if not exists
	_, err = DB.Exec(ctx, `
		DROP TRIGGER IF EXISTS update_conversation_updated_at_trigger ON messages;
		CREATE TRIGGER update_conversation_updated_at_trigger
		AFTER INSERT ON messages
		FOR EACH ROW
		EXECUTE FUNCTION update_conversation_updated_at();
	`)
	if err != nil {
		log.Printf("Error creating trigger: %v", err)
	}

	log.Printf("Database schema initialized")
}

// EnsureUser creates a user if they don't exist
func EnsureUser(ctx context.Context, userID string) error {
	_, err := DB.Exec(ctx, `
		INSERT INTO users (id)
		VALUES ($1)
		ON CONFLICT (id) DO NOTHING
	`, userID)
	return err
}
