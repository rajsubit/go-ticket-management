// Package config loads application and database settings from environment variables and local .env files.
//
// LEARNING GO: Configuration Management & Cloud Deployment
// 1. DATABASE_URL Support:
//    - Cloud platforms like Render, Railway, and Heroku automatically provide a single unified connection URI:
//      `postgres://user:password@host:port/dbname`
//    - We parse this URL using standard library `net/url`, extracting host, port, credentials, and database.
// 2. Precedence:
//    - System Environment (e.g. Render's DATABASE_URL) > Local .env > Default fallbacks.
package config

import (
	"bufio"
	"net/url"
	"os"
	"strings"
	"time"
)

// DBConfig holds PostgreSQL connection settings loaded from DATABASE_URL or individual variables.
type DBConfig struct {
	URL             string
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

	port := getEnv("PORT", "8080")
	dbType := getEnv("DB_TYPE", "postgres")

	// 1. Check for unified DATABASE_URL (Standard on Render, Railway, Heroku)
	if rawURL := getEnv("DATABASE_URL", ""); rawURL != "" {
		if parsedURL, err := url.Parse(rawURL); err == nil {
			host := parsedURL.Hostname()
			dbPort := parsedURL.Port()
			if dbPort == "" {
				dbPort = "5432"
			}

			user := parsedURL.User.Username()
			password, _ := parsedURL.User.Password()
			dbName := strings.TrimPrefix(parsedURL.Path, "/")
			sslMode := parsedURL.Query().Get("sslmode")
			if sslMode == "" {
				sslMode = "require" // Cloud databases typically require SSL
			}

			return Config{
				Port:   port,
				DBType: dbType,
				DB: DBConfig{
					URL:             rawURL,
					Host:            host,
					Port:            dbPort,
					User:            user,
					Password:        password,
					DBName:          dbName,
					SSLMode:         sslMode,
					MaxOpenConns:    25,
					MaxIdleConns:    10,
					ConnMaxLifetime: 15 * time.Minute,
				},
			}
		}
	}

	// 2. Fall back to individual DB_* environment variables
	return Config{
		Port:   port,
		DBType: dbType,
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
