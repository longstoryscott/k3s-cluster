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

	Auth struct {
		JWKSURI string `yaml:"jwks_uri"`
	} `yaml:"auth"`

	Summarization struct {
		MessagesBeforeSummary        int     `yaml:"messages_before_summary"`
		SummariesBeforeConsolidation int     `yaml:"summaries_before_consolidation"`
		SummaryModel                 string  `yaml:"summary_model"`
		SystemPrompt                 string  `yaml:"system_prompt"`
		MaxSummaryLevels             int     `yaml:"max_summary_levels"`         // Maximum depth of summary hierarchy
		SummaryWeightCoefficient     float64 `yaml:"summary_weight_coefficient"` // Weight reduction factor for deeper summaries
		MasterSummaryPrompt          string  `yaml:"master_summary_prompt"`      // Prompt for the master summary of summaries
	} `yaml:"summarization"`
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

	// Auth defaults
	cfg.Auth.JWKSURI = "http://localhost:9091/dex/keys"

	// Summarization defaults
	cfg.Summarization.MessagesBeforeSummary = 10
	cfg.Summarization.SummariesBeforeConsolidation = 5
	cfg.Summarization.SummaryModel = "qwen3:0.6b" // Set default model to qwen3:0.6b
	cfg.Summarization.SystemPrompt = "Summarize the conversation so far in a concise paragraph. Include key points and conclusions, but omit redundant details. The summary will be used as context for future interaction. It should be as small as possible and does not need to be human readable."
	cfg.Summarization.MaxSummaryLevels = 3           // Default to 3 levels of summary depth
	cfg.Summarization.SummaryWeightCoefficient = 0.7 // Each level gets 70% of the weight of the level below
	cfg.Summarization.MasterSummaryPrompt = "Create a comprehensive summary of the conversation, giving most weight to the most recent points and gradually less weight to older information. This is a master summary that will be used for long-term context. It should be as small as possible and does not need to be human readable."
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
	if systemPrompt := os.Getenv("SUMMARIZATION_SYSTEM_PROMPT"); systemPrompt != "" {
		cfg.Summarization.SystemPrompt = systemPrompt
	}

	// Set the global SummaryModel variable
	SummaryModel = cfg.Summarization.SummaryModel
}
