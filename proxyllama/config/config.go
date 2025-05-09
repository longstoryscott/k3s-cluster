// Package config provides configuration handling
package config

import (
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// SummaryModel is a global variable for the summary model name
var SummaryModel string

// CritiqueModel is a global variable for the critique model name
var CritiqueModel string

// EmbeddingModel is a global variable for the embedding model name
var EmbeddingModel string

type SummarizationConfig struct {
	MessagesBeforeSummary        int     `yaml:"messages_before_summary"`
	SummariesBeforeConsolidation int     `yaml:"summaries_before_consolidation"`
	SummaryModel                 string  `yaml:"summary_model"`
	CritiqueModel                string  `yaml:"critique_model"`      // Model to use for response critique
	EmbeddingModel               string  `yaml:"embedding_model"`     // Model to use for embeddings
	EmbeddingDimension           int     `yaml:"embedding_dimension"` // Dimension of embeddings
	EnableRAG                    bool    `yaml:"enable_rag"`          // Enable RAG features
	SystemPrompt                 string  `yaml:"system_prompt"`
	MaxSummaryLevels             int     `yaml:"max_summary_levels"`         // Maximum depth of summary hierarchy
	SummaryWeightCoefficient     float64 `yaml:"summary_weight_coefficient"` // Weight reduction factor for deeper summaries
	MasterSummaryPrompt          string  `yaml:"master_summary_prompt"`      // Prompt for the master summary of summaries
	BriefSystemPrompt            string  `json:"brief_system_prompt"`
	KeyPointsSystemPrompt        string  `json:"key_points_system_prompt"`
	EnableResponseFiltering      bool    `yaml:"enable_response_filtering"` // Enable basic response filtering
	EnableResponseCritique       bool    `yaml:"enable_response_critique"`  // Enable self-critique of responses
}

type RetrievalConfig struct {
	Enabled                 bool    `yaml:"enabled"`                   // Enable memory retrieval
	Limit                   int     `yaml:"limit"`                     // Max number of memories to retrieve
	EnableCrossConversation bool    `yaml:"enable_cross_conversation"` // Allow retrieving memories from other conversations
	SimilarityThreshold     float32 `yaml:"similarity_threshold"`      // Minimum similarity threshold (0.0-1.0)
	AlwaysRetrieve          bool    `yaml:"always_retrieve"`           // Always try to retrieve memories
}

type Config struct {
	Server struct {
		Host string `yaml:"host"`
		Port int    `yaml:"port"`
	} `yaml:"server"`

	Ollama struct {
		BaseURL string `yaml:"base_url"`
	} `yaml:"ollama"`

	Database struct {
		Host           string        `yaml:"host"`
		Port           int           `yaml:"port"`
		User           string        `yaml:"user"`
		Password       string        `yaml:"password"`
		DBName         string        `yaml:"dbname"`
		SSLMode        string        `yaml:"sslmode"`
		ConnectTimeout time.Duration `yaml:"connect_timeout"` // Timeout for database connection
	} `yaml:"database"`

	Redis struct {
		Host               string        `yaml:"host"`
		Port               int           `yaml:"port"`
		Password           string        `yaml:"password"`
		DB                 int           `yaml:"db"`
		ConversationTTL    time.Duration `yaml:"conversation_ttl"`     // TTL for conversation objects
		MessageTTL         time.Duration `yaml:"message_ttl"`          // TTL for message objects
		SummaryTTL         time.Duration `yaml:"summary_ttl"`          // TTL for summary objects
		PoolSize           int           `yaml:"pool_size"`            // Connection pool size
		MinIdleConnections int           `yaml:"min_idle_connections"` // Minimum idle connections in pool
		ConnectTimeout     time.Duration `yaml:"connect_timeout"`      // Timeout for redis connection
		Enabled            bool          `yaml:"enabled"`              // Whether Redis caching is enabled
	} `yaml:"redis"`

	Auth struct {
		JWKSURI string `yaml:"jwks_uri"`
	} `yaml:"auth"`

	Summarization SummarizationConfig `yaml:"summarization"`

	Retrieval RetrievalConfig `yaml:"retrieval"`
}

var (
	config     Config
	configOnce sync.Once
)

// GetConfig loads configuration from config.yaml with environment variable overrides
func GetConfig() Config {
	// Use sync.Once to ensure config is loaded only once
	configOnce.Do(func() {
		// Set defaults first
		setDefaults(&config)

		// Try to read from config file
		if err := loadFromFile(&config); err != nil {
			log.Printf("Warning: couldn't load config file: %v, using defaults and environment variables", err)
		}

		// Override with environment variables
		applyEnvironmentOverrides(&config)
	})

	return config
}

// setDefaults initializes the config with default values
func setDefaults(cfg *Config) {
	// Server defaults
	cfg.Server.Host = "0.0.0.0"
	cfg.Server.Port = 8080

	// Ollama defaults
	cfg.Ollama.BaseURL = "http://localhost:11434"

	// Database defaults
	cfg.Database.Host = "localhost"
	cfg.Database.Port = 5432
	cfg.Database.User = "postgres"
	cfg.Database.Password = "postgres"
	cfg.Database.DBName = "proxyllama"
	cfg.Database.SSLMode = "disable"
	cfg.Database.ConnectTimeout = 10 * time.Second // Default 10-second timeout

	// Redis defaults
	cfg.Redis.Host = "localhost"
	cfg.Redis.Port = 6379
	cfg.Redis.Password = ""
	cfg.Redis.DB = 0
	cfg.Redis.ConversationTTL = 30 * time.Minute
	cfg.Redis.MessageTTL = 1 * time.Hour
	cfg.Redis.SummaryTTL = 2 * time.Hour
	cfg.Redis.PoolSize = 10
	cfg.Redis.MinIdleConnections = 3
	cfg.Redis.ConnectTimeout = 5 * time.Second
	cfg.Redis.Enabled = true

	// Auth defaults
	cfg.Auth.JWKSURI = "http://localhost:9091/dex/keys"

	// Summarization defaults
	cfg.Summarization.MessagesBeforeSummary = 10
	cfg.Summarization.SummariesBeforeConsolidation = 5
	cfg.Summarization.SummaryModel = "qwen3:0.6b"   // Set default model to qwen3:0.6b
	cfg.Summarization.CritiqueModel = "qwen3:0.6b"  // Use a small model for critiques by default
	cfg.Summarization.EmbeddingModel = "qwen3:0.6b" // Set default embedding model to qwen3:0.6b
	cfg.Summarization.EmbeddingDimension = 768      // Default embedding dimension
	cfg.Summarization.EnableRAG = false             // Disable RAG features by default
	cfg.Summarization.SystemPrompt = "Summarize the conversation so far in a concise paragraph. Include key points and conclusions, but omit redundant details. The summary will be used as context for future interaction. It should be as small as possible and does not need to be human readable."
	cfg.Summarization.BriefSystemPrompt = "Create a very concise summary of these short messages. Focus only on essential information and be extremely brief."
	cfg.Summarization.KeyPointsSystemPrompt = "Extract and list the key points from these detailed messages. Identify the main ideas and important details, organizing them in a clear structure."
	cfg.Summarization.MaxSummaryLevels = 3           // Default to 3 levels of summary depth
	cfg.Summarization.SummaryWeightCoefficient = 0.7 // Each level gets 70% of the weight of the level below
	cfg.Summarization.MasterSummaryPrompt = "Create a comprehensive summary of the conversation, giving most weight to the most recent points and gradually less weight to older information. This is a master summary that will be used for long-term context. It should be as small as possible and does not need to be human readable."
	cfg.Summarization.EnableResponseFiltering = true // Enable basic response filtering by default
	cfg.Summarization.EnableResponseCritique = false // Disable response critique by default (more resource intensive)

	// Retrieval defaults
	cfg.Retrieval.Enabled = true                  // Enable memory retrieval by default
	cfg.Retrieval.Limit = 5                       // Retrieve up to 5 memories by default
	cfg.Retrieval.EnableCrossConversation = false // Don't retrieve from other conversations by default
	cfg.Retrieval.SimilarityThreshold = 0.7       // Default similarity threshold of 70%
	cfg.Retrieval.AlwaysRetrieve = false          // Only retrieve when relevant by default
}

// loadFromFile loads configuration from config.yaml
func loadFromFile(cfg *Config) error {
	// Try to read the correct config file based on environment
	var filePath string
	if os.Getenv("LOCAL") == "true" {
		filePath = ".config.local.yaml"
	} else {
		filePath = ".config.yaml"
	}

	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Parse YAML
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(cfg); err != nil {
		return err
	}

	return nil
}

// applyEnvironmentOverrides overrides config with environment variables
func applyEnvironmentOverrides(cfg *Config) {
	// Server overrides
	if host := os.Getenv("SERVER_HOST"); host != "" {
		cfg.Server.Host = host
	}
	if portStr := os.Getenv("SERVER_PORT"); portStr != "" {
		if port, err := strconv.Atoi(portStr); err == nil {
			cfg.Server.Port = port
		} else {
			log.Printf("Warning: invalid SERVER_PORT environment variable: %s", portStr)
		}
	}

	// Ollama overrides
	if ollamaURL := os.Getenv("OLLAMA_URL"); ollamaURL != "" {
		cfg.Ollama.BaseURL = ollamaURL
	}

	// Database overrides
	if dbHost := os.Getenv("DATABASE_HOST"); dbHost != "" {
		cfg.Database.Host = dbHost
	}
	if dbPortStr := os.Getenv("DATABASE_PORT"); dbPortStr != "" {
		if dbPort, err := strconv.Atoi(dbPortStr); err == nil {
			cfg.Database.Port = dbPort
		} else {
			log.Printf("Warning: invalid DATABASE_PORT environment variable: %s", dbPortStr)
		}
	}
	if dbUser := os.Getenv("DATABASE_USER"); dbUser != "" {
		cfg.Database.User = dbUser
	}
	if dbPassword := os.Getenv("DATABASE_PASSWORD"); dbPassword != "" {
		cfg.Database.Password = dbPassword
	}
	if dbName := os.Getenv("DATABASE_DBNAME"); dbName != "" {
		cfg.Database.DBName = dbName
	}
	if dbSSLMode := os.Getenv("DATABASE_SSLMODE"); dbSSLMode != "" {
		cfg.Database.SSLMode = dbSSLMode
	}
	if dbTimeoutStr := os.Getenv("DATABASE_CONNECT_TIMEOUT"); dbTimeoutStr != "" {
		if timeout, err := strconv.Atoi(dbTimeoutStr); err == nil {
			cfg.Database.ConnectTimeout = time.Duration(timeout) * time.Second
		} else {
			log.Printf("Warning: invalid DATABASE_CONNECT_TIMEOUT environment variable: %s", dbTimeoutStr)
		}
	}

	// Redis overrides
	if redisHost := os.Getenv("REDIS_HOST"); redisHost != "" {
		cfg.Redis.Host = redisHost
	}
	if redisPortStr := os.Getenv("REDIS_PORT"); redisPortStr != "" {
		if redisPort, err := strconv.Atoi(redisPortStr); err == nil {
			cfg.Redis.Port = redisPort
		} else {
			log.Printf("Warning: invalid REDIS_PORT environment variable: %s", redisPortStr)
		}
	}
	if redisPassword := os.Getenv("REDIS_PASSWORD"); redisPassword != "" {
		cfg.Redis.Password = redisPassword
	}
	if redisDBStr := os.Getenv("REDIS_DB"); redisDBStr != "" {
		if redisDB, err := strconv.Atoi(redisDBStr); err == nil {
			cfg.Redis.DB = redisDB
		} else {
			log.Printf("Warning: invalid REDIS_DB environment variable: %s", redisDBStr)
		}
	}
	if redisTTLStr := os.Getenv("REDIS_CONVERSATION_TTL"); redisTTLStr != "" {
		if ttl, err := strconv.Atoi(redisTTLStr); err == nil {
			cfg.Redis.ConversationTTL = time.Duration(ttl) * time.Minute
		}
	}
	if redisMsgTTLStr := os.Getenv("REDIS_MESSAGE_TTL"); redisMsgTTLStr != "" {
		if ttl, err := strconv.Atoi(redisMsgTTLStr); err == nil {
			cfg.Redis.MessageTTL = time.Duration(ttl) * time.Minute
		}
	}
	if redisSummaryTTLStr := os.Getenv("REDIS_SUMMARY_TTL"); redisSummaryTTLStr != "" {
		if ttl, err := strconv.Atoi(redisSummaryTTLStr); err == nil {
			cfg.Redis.SummaryTTL = time.Duration(ttl) * time.Minute
		}
	}
	if redisEnabledStr := os.Getenv("REDIS_ENABLED"); redisEnabledStr != "" {
		cfg.Redis.Enabled = redisEnabledStr == "true" || redisEnabledStr == "1" || redisEnabledStr == "yes"
	}

	// Auth overrides
	if jwksEndpoint := os.Getenv("AUTH_JWKS_URI"); jwksEndpoint != "" {
		cfg.Auth.JWKSURI = jwksEndpoint
	}

	// Summarization overrides
	if msgBeforeSummaryStr := os.Getenv("SUMMARIZATION_MESSAGES_BEFORE_SUMMARY"); msgBeforeSummaryStr != "" {
		if val, err := strconv.Atoi(msgBeforeSummaryStr); err == nil {
			cfg.Summarization.MessagesBeforeSummary = val
		}
	}
	if sumBeforeConsolidationStr := os.Getenv("SUMMARIZATION_SUMMARIES_BEFORE_CONSOLIDATION"); sumBeforeConsolidationStr != "" {
		if val, err := strconv.Atoi(sumBeforeConsolidationStr); err == nil {
			cfg.Summarization.SummariesBeforeConsolidation = val
		}
	}
	if summaryModel := os.Getenv("SUMMARIZATION_SUMMARY_MODEL"); summaryModel != "" {
		cfg.Summarization.SummaryModel = summaryModel
	}
	if critiqueModel := os.Getenv("SUMMARIZATION_CRITIQUE_MODEL"); critiqueModel != "" {
		cfg.Summarization.CritiqueModel = critiqueModel
	}
	if embeddingModel := os.Getenv("SUMMARIZATION_EMBEDDING_MODEL"); embeddingModel != "" {
		cfg.Summarization.EmbeddingModel = embeddingModel
	}
	if embeddingDimensionStr := os.Getenv("SUMMARIZATION_EMBEDDING_DIMENSION"); embeddingDimensionStr != "" {
		if val, err := strconv.Atoi(embeddingDimensionStr); err == nil {
			cfg.Summarization.EmbeddingDimension = val
		}
	}
	if enableRAGStr := os.Getenv("SUMMARIZATION_ENABLE_RAG"); enableRAGStr != "" {
		cfg.Summarization.EnableRAG = enableRAGStr == "true" || enableRAGStr == "1" || enableRAGStr == "yes"
	}
	if systemPrompt := os.Getenv("SUMMARIZATION_SYSTEM_PROMPT"); systemPrompt != "" {
		cfg.Summarization.SystemPrompt = systemPrompt
	}

	// Retrieval overrides
	if retrievalEnabledStr := os.Getenv("RETRIEVAL_ENABLED"); retrievalEnabledStr != "" {
		cfg.Retrieval.Enabled = retrievalEnabledStr == "true" || retrievalEnabledStr == "1" || retrievalEnabledStr == "yes"
	}
	if retrievalLimitStr := os.Getenv("RETRIEVAL_LIMIT"); retrievalLimitStr != "" {
		if val, err := strconv.Atoi(retrievalLimitStr); err == nil {
			cfg.Retrieval.Limit = val
		}
	}
	if crossConvStr := os.Getenv("RETRIEVAL_ENABLE_CROSS_CONVERSATION"); crossConvStr != "" {
		cfg.Retrieval.EnableCrossConversation = crossConvStr == "true" || crossConvStr == "1" || crossConvStr == "yes"
	}
	if thresholdStr := os.Getenv("RETRIEVAL_SIMILARITY_THRESHOLD"); thresholdStr != "" {
		if val, err := strconv.ParseFloat(thresholdStr, 32); err == nil {
			cfg.Retrieval.SimilarityThreshold = float32(val)
		}
	}
	if alwaysRetrieveStr := os.Getenv("RETRIEVAL_ALWAYS_RETRIEVE"); alwaysRetrieveStr != "" {
		cfg.Retrieval.AlwaysRetrieve = alwaysRetrieveStr == "true" || alwaysRetrieveStr == "1" || alwaysRetrieveStr == "yes"
	}

	// Set the global SummaryModel variable
	SummaryModel = cfg.Summarization.SummaryModel
	// Set the global CritiqueModel variable
	CritiqueModel = cfg.Summarization.CritiqueModel
	// Set the global EmbeddingModel variable
	EmbeddingModel = cfg.Summarization.EmbeddingModel
}
