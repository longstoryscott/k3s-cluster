package config

type AppConfig struct {
	Port     string `json:"port" mapstructure:"port"`
	Host     string `json:"host" mapstructure:"host"`
	LogLevel string `json:"logLevel" mapstructure:"log_level"`
}

// GetAppConfig returns application configuration from environment variables with fallbacks
func GetAppConfig() AppConfig {
	return AppConfig{
		Port:     getEnv("USRMGR_API_PORT", "3333"),
		Host:     getEnv("USRMGR_API_HOST", ""),
		LogLevel: getEnv("USRMGR_LOG_LEVEL", "info"),
	}
}
