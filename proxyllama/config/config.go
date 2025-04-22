package config

import (
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server struct {
		Port int    `yaml:"port" required:"true"`
		Host string `yaml:"host" required:"true"`
	} `yaml:"server"`

	Auth struct {
		Issuer   string `yaml:"issuer" required:"true"`
		Audience string `yaml:"audience" required:"true"`
		JWKSURI  string `yaml:"jwks_uri" required:"true"`
	} `yaml:"auth"`

	Ollama struct {
		BaseURL string `yaml:"base_url" required:"true"`
	} `yaml:"ollama"`

	Database struct {
		Host     string `yaml:"host" required:"true"`
		Port     int    `yaml:"port" required:"true"`
		User     string `yaml:"user" required:"true"`
		Password string `yaml:"password" required:"true"`
		DBName   string `yaml:"dbname" required:"true"`
		SSLMode  string `yaml:"sslmode" required:"true"`
	} `yaml:"database"`
}

var conf *Config

func loadConfig(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, &conf)
}

func GetConfig() *Config {
	if conf == nil {
		islocal := os.Getenv("LOCAL")
		if islocal == "true" {
			err := loadConfig(".config.local.yaml")
			if err != nil {
				panic(err)
			}
			return conf
		}
		err := loadConfig(".config.yaml")
		if err != nil {
			panic(err)
		}
	}
	return conf
}
