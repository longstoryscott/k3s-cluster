package storage

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"proxyllama/config"
	"proxyllama/models"
	"runtime"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
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

// InitDB initializes the database connection
func InitDB(connStr string) error {
	var err error
	var config *pgxpool.Config
	connString = connStr

	initOnce.Do(func() {
		_, file, line, _ := runtime.Caller(0)
		LoadQueries()
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Info("Initializing PostgreSQL database connection...")

		config, err = pgxpool.ParseConfig(connStr)
		if err != nil {
			err = fmt.Errorf("unable to parse postgres connection string: %w", err)
			return
		}

		// Build the connection pool
		Pool, err = pgxpool.NewWithConfig(context.Background(), config)
		if err != nil {
			err = fmt.Errorf("failed to create connection pool: %w", err)
			return
		}

		// Ping the database to verify the connection
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err = Pool.Ping(ctx); err != nil {
			err = fmt.Errorf("failed to ping database: %w", err)
			return
		}

		// Initialize all the tables
		ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		err = InitializeTables(ctx)
		if err != nil {
			err = fmt.Errorf("failed to initialize tables: %w", err)
			return
		}
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Info("Successfully initialized tables")

		// Ensure research tables are created
		err = EnsureResearchTables(ctx)
		if err != nil {
			err = fmt.Errorf("failed to ensure research tables: %w", err)
			return
		}
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Info("Successfully ensured research tables")

		// Create default model profiles
		err = CreateDefaultProfiles(ctx)
		if err != nil {
			err = fmt.Errorf("failed to create default model profiles: %w", err)
			return
		}

		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Info("Successfully connected to PostgreSQL")
	})
	return err
}

// EnsureDBConnection checks the database connection and attempts to reconnect if necessary
func EnsureDBConnection(ctx context.Context) error {
	if Pool == nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Info("Connection pool is nil, initializing database")

		if connString == "" {
			return errors.New("connection string is empty, cannot reconnect")
		}
		return InitDB(connString)
	}

	// Check if the connection is still valid
	if err := Pool.Ping(ctx); err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Database connection lost, attempting to reconnect")

		// Close the old pool
		Pool.Close()

		// Reinitialize
		if connString == "" {
			return errors.New("connection string is empty, cannot reconnect")
		}
		return InitDB(connString)
	}

	return nil
}

// InitializeTables creates all necessary database tables
func InitializeTables(ctx context.Context) error {
	// Create extensions
	_, err := Pool.Exec(ctx, GetQuery("schema.create_extensions"))
	if err != nil {
		return fmt.Errorf("failed to create extensions: %w", err)
	}

	// Create users table
	_, err = Pool.Exec(ctx, GetQuery("schema.create_users_table"))
	if err != nil {
		return fmt.Errorf("failed to create users table: %w", err)
	}

	// Create conversations table
	_, err = Pool.Exec(ctx, GetQuery("schema.create_conversations_table"))
	if err != nil {
		return fmt.Errorf("failed to create conversations table: %w", err)
	}

	// Create messages table
	_, err = Pool.Exec(ctx, GetQuery("schema.create_messages_table"))
	if err != nil {
		return fmt.Errorf("failed to create messages table: %w", err)
	}

	// Create messages indexes
	_, err = Pool.Exec(ctx, GetQuery("schema.create_messages_indexes"))
	if err != nil {
		return fmt.Errorf("failed to create message indexes: %w", err)
	}

	// Create message embeddings table
	_, err = Pool.Exec(ctx, GetQuery("schema.create_message_embeddings_table"))
	if err != nil {
		return fmt.Errorf("failed to create message embeddings table: %w", err)
	}

	// Create hypertable for message_embeddings
	_, err = Pool.Exec(ctx, GetQuery("schema.create_message_embeddings_hypertable"))
	if err != nil {
		return fmt.Errorf("failed to create message embeddings hypertable: %w", err)
	}

	// Enable compression for message_embeddings
	_, err = Pool.Exec(ctx, GetQuery("schema.enable_message_embeddings_compression"))
	if err != nil {
		return fmt.Errorf("failed to enable message embeddings compression: %w", err)
	}

	// Add compression policy for message_embeddings
	_, err = Pool.Exec(ctx, GetQuery("schema.message_embeddings_compression_policy"))
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Warning: Failed to add message embeddings compression policy")
	}

	// Add retention policy for message_embeddings
	_, err = Pool.Exec(ctx, GetQuery("schema.message_embeddings_retention_policy"))
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Warning: Failed to add message embeddings retention policy")
	}

	// Create summaries table
	_, err = Pool.Exec(ctx, GetQuery("schema.create_summaries_table"))
	if err != nil {
		return fmt.Errorf("failed to create summaries table: %w", err)
	}

	// Create model profiles table
	_, err = Pool.Exec(ctx, GetQuery("schema.create_model_profiles_table"))
	if err != nil {
		return fmt.Errorf("failed to create model profiles table: %w", err)
	}

	// Create model profiles index
	_, err = Pool.Exec(ctx, GetQuery("schema.create_model_profiles_index"))
	if err != nil {
		return fmt.Errorf("failed to create model profiles index: %w", err)
	}

	// Create hypertable for conversations
	_, err = Pool.Exec(ctx, GetQuery("schema.create_conversations_hypertable"))
	if err != nil {
		return fmt.Errorf("failed to create conversations hypertable: %w", err)
	}

	// Create conversations indexes
	_, err = Pool.Exec(ctx, GetQuery("schema.create_conversations_indexes"))
	if err != nil {
		return fmt.Errorf("failed to create conversations indexes: %w", err)
	}

	// Create hypertable for messages
	_, err = Pool.Exec(ctx, GetQuery("schema.create_messages_hypertable"))
	if err != nil {
		return fmt.Errorf("failed to create messages hypertable: %w", err)
	}

	// Create additional message indexes
	_, err = Pool.Exec(ctx, GetQuery("schema.create_additional_messages_indexes"))
	if err != nil {
		return fmt.Errorf("failed to create additional message indexes: %w", err)
	}

	// Create hypertable for summaries
	_, err = Pool.Exec(ctx, GetQuery("schema.create_summaries_hypertable"))
	if err != nil {
		return fmt.Errorf("failed to create summaries hypertable: %w", err)
	}

	// Create summaries indexes
	_, err = Pool.Exec(ctx, GetQuery("schema.create_summaries_indexes"))
	if err != nil {
		return fmt.Errorf("failed to create summaries indexes: %w", err)
	}

	// Create user check trigger
	_, err = Pool.Exec(ctx, GetQuery("schema.create_user_check_trigger"))
	if err != nil {
		return fmt.Errorf("failed to create user check trigger: %w", err)
	}

	// Create conversation update trigger
	_, err = Pool.Exec(ctx, GetQuery("schema.create_conversation_update_trigger"))
	if err != nil {
		return fmt.Errorf("failed to create conversation update trigger: %w", err)
	}

	// Create conversation check triggers
	_, err = Pool.Exec(ctx, GetQuery("schema.create_conversation_check_triggers"))
	if err != nil {
		return fmt.Errorf("failed to create conversation check triggers: %w", err)
	}

	// Create cascade delete trigger
	_, err = Pool.Exec(ctx, GetQuery("schema.create_cascade_delete_trigger"))
	if err != nil {
		return fmt.Errorf("failed to create cascade delete trigger: %w", err)
	}

	// Enable compression on messages
	_, err = Pool.Exec(ctx, GetQuery("schema.enable_messages_compression"))
	if err != nil {
		return fmt.Errorf("failed to enable messages compression: %w", err)
	}

	// Enable compression on conversations
	_, err = Pool.Exec(ctx, GetQuery("schema.enable_conversations_compression"))
	if err != nil {
		return fmt.Errorf("failed to enable conversations compression: %w", err)
	}

	// Enable compression on summaries
	_, err = Pool.Exec(ctx, GetQuery("schema.enable_summaries_compression"))
	if err != nil {
		return fmt.Errorf("failed to enable summaries compression: %w", err)
	}

	// Add compression policy for messages
	_, err = Pool.Exec(ctx, GetQuery("schema.messages_compression_policy"))
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Warning: Failed to add messages compression policy")
	}

	// Add compression policy for conversations
	_, err = Pool.Exec(ctx, GetQuery("schema.conversations_compression_policy"))
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Warning: Failed to add conversations compression policy")
	}

	// Add compression policy for summaries
	_, err = Pool.Exec(ctx, GetQuery("schema.summaries_compression_policy"))
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Warning: Failed to add summaries compression policy")
	}

	// Add retention policy for conversations
	_, err = Pool.Exec(ctx, GetQuery("schema.conversations_retention_policy"))
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Warning: Failed to add conversations retention policy")
	}

	// Add retention policy for messages
	_, err = Pool.Exec(ctx, GetQuery("schema.messages_retention_policy"))
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Warning: Failed to add messages retention policy")
	}

	// Add retention policy for summaries
	_, err = Pool.Exec(ctx, GetQuery("schema.summaries_retention_policy"))
	if err != nil {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Warn("Warning: Failed to add summaries retention policy")
	}

	// Initialize memory schema
	InitMemorySchema(ctx)

	return nil
}

// EnsureResearchTables ensures that the research tables are created
func EnsureResearchTables(ctx context.Context) error {
	// Create research_tasks table
	_, err := Pool.Exec(ctx, GetQuery("schema.create_research_tasks_table"))
	if err != nil {
		return fmt.Errorf("failed to create research_tasks table: %w", err)
	}

	// Create research_subtasks table
	_, err = Pool.Exec(ctx, GetQuery("schema.create_research_subtasks_table"))
	if err != nil {
		return fmt.Errorf("failed to create research_subtasks table: %w", err)
	}

	return nil
}

// CreateDefaultProfiles creates the default model profiles in the database
func CreateDefaultProfiles(ctx context.Context) error {
	// Otherwise, insert the default profiles
	defaultProfiles := []models.ModelProfile{
		config.DefaultPrimaryProfile,
		config.DefaultSummarizationProfile,
		config.DefaultMasterSummaryProfile,
		config.DefaultBriefSummaryProfile,
		config.DefaultKeyPointsProfile,
		config.DefaultSelfCritiqueProfile,
		config.DefaultImprovementProfile,
		config.DefaultMemoryRetrievalProfile,
		config.DefaultAnalysisProfile,
		config.DefaultResearchTaskProfile,
		config.DefaultResearchPlanProfile,
		config.DefaultResearchConsolidationProfile,
		config.DefaultResearchAnalysisProfile,
		config.DefaultEmbeddingProfile,
	}

	tx, err := Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	// Insert each default profile
	systemUserID := "0"
	for _, profile := range defaultProfiles {
		_, file, line, _ := runtime.Caller(0)
		logrus.WithFields(logrus.Fields{
			"file":        filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":        line,
			"profileName": profile.Name,
			"profileId":   profile.ID,
		}).Info("Inserting default profile")

		_, err = tx.Exec(ctx,
			GetQuery("modelprofile.create_default_profile"),
			profile.ID.String(),
			systemUserID,
			profile.Name,
			profile.Description,
			profile.ModelName,
			profile.Parameters,
			profile.SystemPrompt,
			profile.ModelVersion,
			profile.Type,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// PerformDatabaseMaintenance runs optimization and maintenance tasks for the database
func PerformDatabaseMaintenance(ctx context.Context) error {
	_, file, line, _ := runtime.Caller(0)
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Starting database maintenance tasks...")

	// Vacuum analyze for better query planning
	_, err := Pool.Exec(ctx, `VACUUM ANALYZE`)
	if err != nil {
		return fmt.Errorf("failed to vacuum analyze: %w", err)
	}
	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("VACUUM ANALYZE completed")

	// Reindex tables to optimize indexes
	for _, table := range []string{"messages", "conversations", "summaries"} {
		_, err = Pool.Exec(ctx, fmt.Sprintf("REINDEX TABLE %s", table))
		if err != nil {
			return fmt.Errorf("failed to reindex %s: %w", table, err)
		}
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"table": table,
		}).Info("REINDEX completed")
	}

	// Run TimescaleDB-specific maintenance
	_, err = Pool.Exec(ctx, `SELECT run_job(j.id) FROM timescaledb_information.jobs j WHERE j.proc_name = 'policy_refresh'`)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"file":  filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line":  line,
			"error": err,
		}).Info("Note: TimescaleDB policy refresh failed (may be normal if no jobs)")
	} else {
		logrus.WithFields(logrus.Fields{
			"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
			"line": line,
		}).Info("TimescaleDB policy refresh completed")
	}

	logrus.WithFields(logrus.Fields{
		"file": filepath.Join(filepath.Base(filepath.Dir(file)), filepath.Base(file)),
		"line": line,
	}).Info("Database maintenance tasks completed successfully")
	return nil
}

// EnsureUser creates a user if they don't exist
func EnsureUser(ctx context.Context, userID string) error {
	_, err := Pool.Exec(ctx, GetQuery("user.ensure_user"), userID)
	return err
}
