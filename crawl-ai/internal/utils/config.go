package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

// Config represents the application configuration
type Config struct {
	Language    string
	WikiDir     string
	AIModel     string
	AIBaseURL   string
	AIProvider  string
	SearchBaseURL string
	Project     string
}

// LoadConfig initializes the configuration with precedence:
// Flags > Env Vars > Project Config > Global Config
func LoadConfig(project string) (*Config, error) {
	v := viper.New()

	// 1. Set Defaults
	v.SetDefault("language", "ko")
	v.SetDefault("out", "./wiki")
	v.SetDefault("model", "llama3")
	v.SetDefault("ai_provider", "ollama")
	v.SetDefault("ai_url", "http://localhost:11434/v1")
	v.SetDefault("search_base_url", "")

	// 2. Global Config (~/.musu/config.toml)
	home, _ := os.UserHomeDir()
	globalDir := filepath.Join(home, ".musu")
	if err := os.MkdirAll(globalDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create global config dir: %v", err)
	}

	v.SetConfigName("config")
	v.SetConfigType("toml")
	v.AddConfigPath(globalDir)
	v.ReadInConfig() // Ignore error if not found

	// 3. Project Config (wiki/projects/{project}/config.toml)
	wikiDir := v.GetString("out")
	if project != "" && project != "all" && project != "default" {
		projectDir := filepath.Join(wikiDir, "projects", project)
		if _, err := os.Stat(projectDir); err == nil {
			v.AddConfigPath(projectDir)
			v.MergeInConfig()
		}

		// 4. Project Secrets (.env)
		envPath := filepath.Join(projectDir, ".env")
		if _, err := os.Stat(envPath); err == nil {
			godotenv.Load(envPath)
		}
	}

	// 5. Environment Variables (MUSU_*)
	v.SetEnvPrefix("MUSU")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	return &Config{
		Language:   v.GetString("language"),
		WikiDir:    v.GetString("out"),
		AIModel:    v.GetString("model"),
		AIProvider: firstNonEmpty(viper.GetString("ai_provider"), v.GetString("ai_provider")),
		AIBaseURL:  firstNonEmpty(viper.GetString("ai_url"), v.GetString("ai_url")),
		SearchBaseURL: v.GetString("search_base_url"),
		Project:    project,
	}, nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// GetSecret retrieves a sensitive value from environment variables
func GetSecret(key string) string {
	return os.Getenv(key)
}
