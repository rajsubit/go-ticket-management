// Package config loads application and database settings from environment variables and local .env files.
//
// LEARNING GO: Configuration Management & Security
// 1. Never hardcode credentials in source code:
//    - Passwords, usernames, and database names should be passed via environment variables.
//    - This allows identical code to run in development, testing, staging, and production securely.
// 2. The Twelve-Factor App Principle:
//    - Store config in the environment.
// 3. Simple .env Loader:
//    - We use Go's standard library (`bufio.Scanner`, `os.Open`) to parse `.env` files line-by-line
//      without needing external heavy dependencies.
package config

import (
	"bufio"
	"os"
	"strings"
	"time"
)

// DBConfig holds PostgreSQL connection settings loaded strictly from the environment.
type DBConfig struct {
	Host            string
	Port            string
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Config holds all combined application settings.
type Config struct {
	Port   string
	DBType string // "postgres" or "memory"
	DB     DBConfig
}

// Load reads any local .env file (if present) and returns the parsed application Config.
// Environment variables already present on the system take precedence over .env file values.
func Load() Config {
	// Attempt to load .env file from working directory, or parent directory for tests
	if !loadDotEnv(".env") {
		_ = loadDotEnv("../.env")
	}

	return Config{
		Port:   getEnv("PORT", "8080"),
		DBType: getEnv("DB_TYPE", "postgres"),
		DB: DBConfig{
			Host:            getEnv("DB_HOST", "localhost"),
			Port:            getEnv("DB_PORT", "5432"),
			User:            getEnv("DB_USER", ""),
			Password:        getEnv("DB_PASSWORD", ""),
			DBName:          getEnv("DB_NAME", ""),
			SSLMode:         getEnv("DB_SSLMODE", "disable"),
			MaxOpenConns:    25,
			MaxIdleConns:    10,
			ConnMaxLifetime: 15 * time.Minute,
		},
	}
}

// getEnv retrieves an environment variable or falls back to a default value.
func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// loadDotEnv parses a .env file and sets variables with os.Setenv if not already set.
// Returns true if the file was found and processed, false otherwise.
func loadDotEnv(filepath string) bool {
	file, err := os.Open(filepath)
	if err != nil {
		return false // File does not exist
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip blank lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Split into KEY=VALUE
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		// Strip optional quotes
		val = strings.Trim(val, `"'`)

		// Only set if not already defined in system environment
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
	return true
}
