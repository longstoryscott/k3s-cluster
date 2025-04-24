package storage

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

var (
	Pool       *pgxpool.Pool
	initOnce   sync.Once = sync.Once{}
	connString string    // Store connection string for reconnection logic
)

// InitDB initializes the database connection with dynamic pool settings
func InitDB(connStr string) error {
	var err error
	connString = connStr // Store for potential reconnection

	initOnce.Do(func() {
		// Create a connection pool config with optimized settings
		config, e := pgxpool.ParseConfig(connString)
		if e != nil {
			err = fmt.Errorf("unable to parse database config: %w", e)
			return
		}

		// Get connection pool settings from environment or use defaults
		maxConns := 10
		minConns := 2
		if envMaxConns := os.Getenv("DB_MAX_CONNECTIONS"); envMaxConns != "" {
			if val, e := strconv.Atoi(envMaxConns); e == nil && val > 0 {
				maxConns = val
			}
		}

		if envMinConns := os.Getenv("DB_MIN_CONNECTIONS"); envMinConns != "" {
			if val, e := strconv.Atoi(envMinConns); e == nil && val > 0 {
				minConns = val
			}
		}

		// Configure connection pool with best practices
		config.MaxConns = int32(maxConns)
		config.MinConns = int32(minConns)
		config.MaxConnLifetime = 1 * time.Hour     // Prevent stale connections
		config.MaxConnIdleTime = 30 * time.Minute  // Release idle connections
		config.HealthCheckPeriod = 1 * time.Minute // Regular health checks

		// Create the connection pool
		Pool, e = pgxpool.NewWithConfig(context.Background(), config)
		if e != nil {
			err = fmt.Errorf("unable to connect to database: %w", e)
			return
		}

		// Verify connection works
		if e := Pool.Ping(context.Background()); e != nil {
			err = fmt.Errorf("unable to ping database: %w", e)
			return
		}

		log.Println("Successfully connected to database with connection pool")

		// Schedule periodic health checks
		go startBackgroundHealthChecks()
	})
	return err
}

// EnsureDBConnection checks the database connection and attempts to reconnect if necessary
func EnsureDBConnection(ctx context.Context) error {
	if Pool == nil {
		return errors.New("database connection not initialized")
	}

	err := Pool.Ping(ctx)
	if err != nil {
		log.Printf("Database ping failed: %v, attempting reconnection", err)

		// Try to close existing connections
		if Pool != nil {
			Pool.Close()
		}

		// Clear init flag to allow reconnection
		initOnce = sync.Once{}

		// Reinitialize connection
		return InitDB(connString)
	}

	return nil
}

// startBackgroundHealthChecks periodically checks database connection health
func startBackgroundHealthChecks() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := EnsureDBConnection(ctx); err != nil {
			log.Printf("Background health check failed: %v", err)
		}
		cancel()

		// Run maintenance tasks occasionally (once a day)
		go func() {
			// Use time-based condition to run maintenance once a day
			hour := time.Now().Hour()
			if hour == 3 { // 3 AM, typically low traffic
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
				defer cancel()
				if err := PerformDatabaseMaintenance(ctx); err != nil {
					log.Printf("Database maintenance failed: %v", err)
				}
			}
		}()
	}
}

// InitSchema creates the database schema if it doesn't exist
func InitSchema(ctx context.Context) {
	// Ensure TimescaleDB extension is available
	_, err := Pool.Exec(ctx, `CREATE EXTENSION IF NOT EXISTS timescaledb CASCADE;`)
	if err != nil {
		log.Printf("Warning: TimescaleDB extension could not be enabled: %v", err)
		log.Printf("Continuing without TimescaleDB hypertable support")
	}

	// Create users table
	_, err = Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		log.Printf("Error creating users table: %v", err)
	}

	// Create conversations table with TimescaleDB compatible schema
	_, err = Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS conversations (
			id SERIAL,
			user_id TEXT NOT NULL,
			title TEXT DEFAULT 'New conversation',
			model TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (id, created_at)
		)
	`)
	if err != nil {
		log.Printf("Error creating conversations table: %v", err)
	}

	// Create messages table with TimescaleDB compatible schema
	_, err = Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS messages (
			id SERIAL,
			conversation_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (id, created_at)
		)
	`)
	if err != nil {
		log.Printf("Error creating messages table: %v", err)
	}

	// Create summaries table with TimescaleDB compatible schema
	_, err = Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS summaries (
			id SERIAL,
			conversation_id INTEGER NOT NULL,
			content TEXT NOT NULL,
			level INTEGER NOT NULL,
			source_ids JSONB NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			PRIMARY KEY (id, created_at)
		)
	`)
	if err != nil {
		log.Printf("Error creating summaries table: %v", err)
	}

	// Create hypertable for conversations with optimal chunk interval
	_, err = Pool.Exec(ctx, `
		SELECT create_hypertable('conversations', 'created_at', 
							   if_not_exists => TRUE, 
							   migrate_data => TRUE,
							   chunk_time_interval => INTERVAL '7 days')
	`)
	if err != nil {
		log.Printf("Note: Could not create hypertable for conversations: %v", err)
	} else {
		log.Printf("Conversations hypertable created or already exists")

		// Create additional indexes for conversations
		_, err = Pool.Exec(ctx, `
			CREATE INDEX IF NOT EXISTS idx_conversations_user_time ON conversations (user_id, created_at DESC);
		`)
		if err != nil {
			log.Printf("Warning: Could not create additional indexes for conversations: %v", err)
		}
	}

	// Create hypertable for messages with optimal chunk interval
	_, err = Pool.Exec(ctx, `
		SELECT create_hypertable('messages', 'created_at', 
							   if_not_exists => TRUE, 
							   migrate_data => TRUE,
							   chunk_time_interval => INTERVAL '3 days')
	`)
	if err != nil {
		log.Printf("Note: Could not create hypertable for messages: %v", err)
	} else {
		log.Printf("Messages hypertable created or already exists")

		// Create additional indexes for messages
		_, err = Pool.Exec(ctx, `
			CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages (conversation_id);
			CREATE INDEX IF NOT EXISTS idx_messages_conversation_time ON messages (conversation_id, created_at DESC);
		`)
		if err != nil {
			log.Printf("Warning: Could not create additional indexes for messages: %v", err)
		}
	}

	// Create hypertable for summaries with optimal chunk interval
	_, err = Pool.Exec(ctx, `
		SELECT create_hypertable('summaries', 'created_at', 
							   if_not_exists => TRUE, 
							   migrate_data => TRUE,
							   chunk_time_interval => INTERVAL '7 days')
	`)
	if err != nil {
		log.Printf("Note: Could not create hypertable for summaries: %v", err)
	} else {
		log.Printf("Summaries hypertable created or already exists")

		// Create additional indexes for summaries
		_, err = Pool.Exec(ctx, `
			CREATE INDEX IF NOT EXISTS idx_summaries_conversation_id ON summaries (conversation_id);
			CREATE INDEX IF NOT EXISTS idx_summaries_conversation_level ON summaries (conversation_id, level);
			CREATE INDEX IF NOT EXISTS idx_summaries_conversation_time ON summaries (conversation_id, created_at DESC);
		`)
		if err != nil {
			log.Printf("Warning: Could not create additional indexes for summaries: %v", err)
		}
	}

	// Instead of foreign keys, create a trigger to check for valid user_id
	_, err = Pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION check_user_exists()
		RETURNS TRIGGER AS $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM users WHERE id = NEW.user_id) THEN
				RAISE EXCEPTION 'Referenced user does not exist';
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS ensure_user_exists_trigger ON conversations;
		CREATE TRIGGER ensure_user_exists_trigger
		BEFORE INSERT OR UPDATE ON conversations
		FOR EACH ROW
		EXECUTE FUNCTION check_user_exists();
	`)
	if err != nil {
		log.Printf("Warning: Could not create user existence check trigger: %v", err)
	}

	// Update conversation updated_at when a message is added
	_, err = Pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION update_conversation_updated_at()
		RETURNS TRIGGER AS $$
		BEGIN
			UPDATE conversations
			SET updated_at = NOW()
			WHERE id = NEW.conversation_id;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS update_conversation_updated_at_trigger ON messages;
		CREATE TRIGGER update_conversation_updated_at_trigger
		AFTER INSERT ON messages
		FOR EACH ROW
		EXECUTE FUNCTION update_conversation_updated_at();
	`)
	if err != nil {
		log.Printf("Error creating update_conversation_updated_at function and trigger: %v", err)
	}

	// Add triggers to maintain referential integrity between conversations and messages
	_, err = Pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION check_conversation_exists()
		RETURNS TRIGGER AS $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM conversations WHERE id = NEW.conversation_id) THEN
				RAISE EXCEPTION 'Referenced conversation does not exist';
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS ensure_conversation_exists_messages_trigger ON messages;
		CREATE TRIGGER ensure_conversation_exists_messages_trigger
		BEFORE INSERT OR UPDATE ON messages
		FOR EACH ROW
		EXECUTE FUNCTION check_conversation_exists();

		DROP TRIGGER IF EXISTS ensure_conversation_exists_summaries_trigger ON summaries;
		CREATE TRIGGER ensure_conversation_exists_summaries_trigger
		BEFORE INSERT OR UPDATE ON summaries
		FOR EACH ROW
		EXECUTE FUNCTION check_conversation_exists();
	`)
	if err != nil {
		log.Printf("Warning: Could not create conversation existence check triggers: %v", err)
	}

	// Add ON DELETE CASCADE functionality via triggers
	_, err = Pool.Exec(ctx, `
		CREATE OR REPLACE FUNCTION delete_related_messages_and_summaries()
		RETURNS TRIGGER AS $$
		BEGIN
			DELETE FROM messages WHERE conversation_id = OLD.id;
			DELETE FROM summaries WHERE conversation_id = OLD.id;
			RETURN OLD;
		END;
		$$ LANGUAGE plpgsql;

		DROP TRIGGER IF EXISTS cascade_delete_trigger ON conversations;
		CREATE TRIGGER cascade_delete_trigger
		BEFORE DELETE ON conversations
		FOR EACH ROW
		EXECUTE FUNCTION delete_related_messages_and_summaries();
	`)
	if err != nil {
		log.Printf("Warning: Could not create cascade delete trigger: %v", err)
	}

	// Enable compression on hypertables before adding compression policies
	_, err = Pool.Exec(ctx, `
	ALTER TABLE messages SET (timescaledb.compress, timescaledb.compress_segmentby = 'conversation_id');
`)
	if err != nil {
		log.Printf("Warning: Could not enable compression for messages: %v", err)
	} else {
		log.Printf("Compression enabled for messages")
	}

	_, err = Pool.Exec(ctx, `
	ALTER TABLE conversations SET (timescaledb.compress, timescaledb.compress_segmentby = 'user_id');
`)
	if err != nil {
		log.Printf("Warning: Could not enable compression for conversations: %v", err)
	} else {
		log.Printf("Compression enabled for conversations")
	}

	_, err = Pool.Exec(ctx, `
	ALTER TABLE summaries SET (timescaledb.compress, timescaledb.compress_segmentby = 'conversation_id');
`)
	if err != nil {
		log.Printf("Warning: Could not enable compression for summaries: %v", err)
	} else {
		log.Printf("Compression enabled for summaries")
	}

	// Add data compression policies for hypertables
	_, err = Pool.Exec(ctx, `
		SELECT add_compression_policy('messages', INTERVAL '7 days', 
								   if_not_exists => TRUE);
	`)
	if err != nil {
		log.Printf("Warning: Could not create compression policy for messages: %v", err)
	} else {
		log.Printf("Compression policy for messages created successfully")
	}

	_, err = Pool.Exec(ctx, `
		SELECT add_compression_policy('conversations', INTERVAL '30 days', 
								   if_not_exists => TRUE);
	`)
	if err != nil {
		log.Printf("Warning: Could not create compression policy for conversations: %v", err)
	} else {
		log.Printf("Compression policy for conversations created successfully")
	}

	_, err = Pool.Exec(ctx, `
		SELECT add_compression_policy('summaries', INTERVAL '14 days', 
								   if_not_exists => TRUE);
	`)
	if err != nil {
		log.Printf("Warning: Could not create compression policy for summaries: %v", err)
	} else {
		log.Printf("Compression policy for summaries created successfully")
	}

	// Add retention policies for hypertables
	_, err = Pool.Exec(ctx, `
		SELECT add_retention_policy('messages', INTERVAL '90 days', 
								 if_not_exists => TRUE);
	`)
	if err != nil {
		log.Printf("Warning: Could not create retention policy for messages: %v", err)
	} else {
		log.Printf("Retention policy for messages created successfully")
	}

	_, err = Pool.Exec(ctx, `
		SELECT add_retention_policy('conversations', INTERVAL '365 days', 
								 if_not_exists => TRUE);
	`)
	if err != nil {
		log.Printf("Warning: Could not create retention policy for conversations: %v", err)
	} else {
		log.Printf("Retention policy for conversations created successfully")
	}

	_, err = Pool.Exec(ctx, `
		SELECT add_retention_policy('summaries', INTERVAL '180 days', 
								 if_not_exists => TRUE);
	`)
	if err != nil {
		log.Printf("Warning: Could not create retention policy for summaries: %v", err)
	} else {
		log.Printf("Retention policy for summaries created successfully")
	}

	log.Printf("Database schema initialized with TimescaleDB optimizations")
}

// PerformDatabaseMaintenance runs optimization and maintenance tasks for the database
func PerformDatabaseMaintenance(ctx context.Context) error {
	log.Printf("Starting database maintenance tasks...")

	// Vacuum analyze for better query planning
	_, err := Pool.Exec(ctx, `VACUUM ANALYZE`)
	if err != nil {
		return fmt.Errorf("failed to vacuum analyze: %w", err)
	}
	log.Printf("VACUUM ANALYZE completed")

	// Reindex tables to optimize indexes
	for _, table := range []string{"messages", "conversations", "summaries"} {
		_, err = Pool.Exec(ctx, fmt.Sprintf("REINDEX TABLE %s", table))
		if err != nil {
			return fmt.Errorf("failed to reindex %s: %w", table, err)
		}
		log.Printf("REINDEX of %s completed", table)
	}

	// Run TimescaleDB-specific maintenance
	_, err = Pool.Exec(ctx, `SELECT run_job(j.id) FROM timescaledb_information.jobs j WHERE j.proc_name = 'policy_refresh'`)
	if err != nil {
		log.Printf("Note: TimescaleDB policy refresh failed (may be normal if no jobs): %v", err)
	} else {
		log.Printf("TimescaleDB policy refresh completed")
	}

	log.Printf("Database maintenance tasks completed successfully")
	return nil
}

// EnsureUser creates a user if they don't exist
func EnsureUser(ctx context.Context, userID string) error {
	_, err := Pool.Exec(ctx, `
		INSERT INTO users (id)
		VALUES ($1)
		ON CONFLICT (id) DO NOTHING
	`, userID)
	return err
}
