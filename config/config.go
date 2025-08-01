// Copyright (c) 2025 Sudo-Ivan
// Licensed under the MIT License

// Package config provides configuration management for the website-archiver application.
// It handles environment variables, default values, and logging setup.
package config

import (
	"fmt"
	"log/slog"
	"os"
	"time"
)

const (
	// DefaultMaxDepth is the default maximum depth for recursive downloads
	DefaultMaxDepth = 5
	// DefaultDirPerms is the default directory permissions in octal
	DefaultDirPerms = 0750
	// DefaultHTTPTimeout is the default timeout for HTTP requests
	DefaultHTTPTimeout = 30 * time.Second
	// DefaultWaybackAPIURL is the default URL for the Wayback Machine CDX API
	DefaultWaybackAPIURL = "https://web.archive.org/cdx/search/cdx"
	// DefaultOutputDir is the default directory for downloaded files
	DefaultOutputDir = "downloads"
	// DefaultFilePerms is the default file permissions in octal
	DefaultFilePerms = 0600
	// EmptyString represents an empty string constant
	EmptyString = ""
)

// Config holds all configuration values for the application
type Config struct {
	// HTTP related settings
	HTTPTimeout time.Duration
	MaxDepth    int
	DirPerms    os.FileMode

	// File permissions
	FilePerms os.FileMode

	// Wayback Machine settings
	WaybackAPIURL string

	// Output settings
	OutputDir string

	// Logging settings
	LogLevel slog.Level
}

// New creates a new Config instance with values from environment variables or defaults
func New() *Config {
	config := &Config{
		HTTPTimeout:   getEnvDuration("HTTP_TIMEOUT", DefaultHTTPTimeout),
		MaxDepth:      getEnvInt("MAX_DEPTH", DefaultMaxDepth),
		DirPerms:      getEnvFileMode("DIR_PERMS", DefaultDirPerms),
		FilePerms:     getEnvFileMode("FILE_PERMS", DefaultFilePerms),
		WaybackAPIURL: getEnvString("WAYBACK_API_URL", DefaultWaybackAPIURL),
		OutputDir:     getEnvString("OUTPUT_DIR", DefaultOutputDir),
		LogLevel:      getEnvLogLevel("LOG_LEVEL", slog.LevelInfo),
	}

	// Configure slog
	opts := &slog.HandlerOptions{
		Level: config.LogLevel,
	}
	handler := slog.NewJSONHandler(os.Stdout, opts)
	logger := slog.New(handler)
	slog.SetDefault(logger)

	return config
}

// Helper functions to get environment variables with defaults
func getEnvString(key, defaultValue string) string {
	if value := os.Getenv(key); value != EmptyString {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != EmptyString {
		var result int
		if _, err := fmt.Sscanf(value, "%d", &result); err == nil {
			return result
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != EmptyString {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getEnvFileMode(key string, defaultValue os.FileMode) os.FileMode {
	if value := os.Getenv(key); value != EmptyString {
		var mode uint32
		if _, err := fmt.Sscanf(value, "%o", &mode); err == nil {
			return os.FileMode(mode)
		}
	}
	return defaultValue
}

func getEnvLogLevel(key string, defaultValue slog.Level) slog.Level {
	if value := os.Getenv(key); value != EmptyString {
		switch value {
		case "DEBUG":
			return slog.LevelDebug
		case "INFO":
			return slog.LevelInfo
		case "WARN":
			return slog.LevelWarn
		case "ERROR":
			return slog.LevelError
		}
	}
	return defaultValue
}
