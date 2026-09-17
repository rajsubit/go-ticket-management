// Package tests contains all test suites organized in a single folder for easy access.
//
// LEARNING GO: Dedicated Test Suite Organization
// All tests in this folder import packages from `ticket-management/internal/...` as an external consumer would.
package tests

import (
	"os"
	"path/filepath"
	"testing"

	"ticket-management/internal/platform/config"
)

func TestConfigLoadWithDefaults(t *testing.T) {
	t.Setenv("PORT", "")
	t.Setenv("DB_TYPE", "")
	t.Setenv("DB_HOST", "")
	t.Setenv("DB_USER", "")

	cfg := config.Load()

	if cfg.Port != "8080" {
		t.Errorf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.DBType != "postgres" {
		t.Errorf("expected default DBType postgres, got %s", cfg.DBType)
	}
	if cfg.DB.Host != "localhost" {
		t.Errorf("expected default DB host localhost, got %s", cfg.DB.Host)
	}
	if cfg.DB.Port != "5432" {
		t.Errorf("expected default DB port 5432, got %s", cfg.DB.Port)
	}
}

func TestConfigLoadFromEnvironment(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("DB_TYPE", "memory")
	t.Setenv("DB_HOST", "db.internal")
	t.Setenv("DB_USER", "custom_user")
	t.Setenv("DB_PASSWORD", "supersecret")
	t.Setenv("DB_NAME", "custom_db")

	cfg := config.Load()

	if cfg.Port != "9090" {
		t.Errorf("expected port 9090, got %s", cfg.Port)
	}
	if cfg.DBType != "memory" {
		t.Errorf("expected DBType memory, got %s", cfg.DBType)
	}
	if cfg.DB.Host != "db.internal" {
		t.Errorf("expected DB host db.internal, got %s", cfg.DB.Host)
	}
	if cfg.DB.User != "custom_user" {
		t.Errorf("expected DB user custom_user, got %s", cfg.DB.User)
	}
	if cfg.DB.Password != "supersecret" {
		t.Errorf("expected DB password supersecret, got %s", cfg.DB.Password)
	}
	if cfg.DB.DBName != "custom_db" {
		t.Errorf("expected DB name custom_db, got %s", cfg.DB.DBName)
	}
}

func TestConfigLoadFromDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://render_user:render_pass@dpg-xxx.oregon-postgres.render.com:5432/render_db?sslmode=require")

	cfg := config.Load()

	if cfg.DB.User != "render_user" {
		t.Errorf("expected DB user render_user, got %s", cfg.DB.User)
	}
	if cfg.DB.Password != "render_pass" {
		t.Errorf("expected DB password render_pass, got %s", cfg.DB.Password)
	}
	if cfg.DB.Host != "dpg-xxx.oregon-postgres.render.com" {
		t.Errorf("expected DB host dpg-xxx.oregon-postgres.render.com, got %s", cfg.DB.Host)
	}
	if cfg.DB.DBName != "render_db" {
		t.Errorf("expected DB name render_db, got %s", cfg.DB.DBName)
	}
	if cfg.DB.SSLMode != "require" {
		t.Errorf("expected SSL mode require, got %s", cfg.DB.SSLMode)
	}
	if cfg.DB.URL == "" {
		t.Errorf("expected DB URL to be preserved, got empty")
	}
}

func TestDotEnvFileParsing(t *testing.T) {
	tmpDir := t.TempDir()
	envPath := filepath.Join(tmpDir, ".env")

	content := `
	# Test comment
	PORT=4000
	DB_USER="env_user"
	DB_NAME='env_db'
	INVALID_LINE_WITHOUT_EQUALS
	`
	if err := os.WriteFile(envPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test .env: %v", err)
	}

	// Capture and clear pre-existing env vars for clean test isolation
	origPort, hasPort := os.LookupEnv("PORT")
	origUser, hasUser := os.LookupEnv("DB_USER")
	origName, hasName := os.LookupEnv("DB_NAME")

	_ = os.Unsetenv("PORT")
	_ = os.Unsetenv("DB_USER")
	_ = os.Unsetenv("DB_NAME")

	originalWd, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	defer func() {
		_ = os.Chdir(originalWd)
		if hasPort {
			_ = os.Setenv("PORT", origPort)
		} else {
			_ = os.Unsetenv("PORT")
		}
		if hasUser {
			_ = os.Setenv("DB_USER", origUser)
		} else {
			_ = os.Unsetenv("DB_USER")
		}
		if hasName {
			_ = os.Setenv("DB_NAME", origName)
		} else {
			_ = os.Unsetenv("DB_NAME")
		}
	}()

	cfg := config.Load()

	if cfg.Port != "4000" {
		t.Errorf("expected port 4000 loaded from .env, got %s", cfg.Port)
	}
	if cfg.DB.User != "env_user" {
		t.Errorf("expected DB user 'env_user' loaded from .env, got %s", cfg.DB.User)
	}
	if cfg.DB.DBName != "env_db" {
		t.Errorf("expected DB name 'env_db' loaded from .env, got %s", cfg.DB.DBName)
	}
}
