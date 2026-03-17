package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ServerPort       string
	InferenceURL     string
	JWTPrivateKey    string
	JWTExpireSeconds int
	RequestTimeout   time.Duration
	DefaultGap       float64
}

func Load() (Config, error) {
	if err := loadDotEnvFiles(); err != nil {
		return Config{}, err
	}

	cfg := Config{
		ServerPort:       getenv("SERVER_PORT", "18080"),
		InferenceURL:     strings.TrimRight(strings.TrimSpace(os.Getenv("INFERENCE_URL")), "/"),
		JWTPrivateKey:    strings.TrimSpace(os.Getenv("INFERENCE_JWT_PRIVATE_KEY")),
		JWTExpireSeconds: getint("INFERENCE_JWT_EXPIRE_SECONDS", 300),
		RequestTimeout:   getduration("INFERENCE_TIMEOUT", 5*time.Minute),
		DefaultGap:       getfloat("DEFAULT_GAP_SECONDS", 0.1),
	}

	if cfg.InferenceURL == "" {
		return Config{}, fmt.Errorf("INFERENCE_URL is required")
	}
	if cfg.DefaultGap < 0 {
		return Config{}, fmt.Errorf("DEFAULT_GAP_SECONDS must be >= 0")
	}

	return cfg, nil
}

func loadDotEnvFiles() error {
	candidates := []string{"../.env", ".env"}
	initialEnv := map[string]bool{}
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if found {
			initialEnv[key] = true
		}
	}

	for _, candidate := range candidates {
		if err := loadDotEnvFile(candidate, initialEnv); err != nil {
			return err
		}
	}
	return nil
}

func loadDotEnvFile(path string, lockedKeys map[string]bool) error {
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve %s: %w", path, err)
	}

	file, err := os.Open(absolutePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("open %s: %w", absolutePath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for lineNumber := 1; scanner.Scan(); lineNumber++ {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "export ") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			return fmt.Errorf("parse %s:%d: missing '='", absolutePath, lineNumber)
		}

		key = strings.TrimSpace(key)
		if key == "" {
			return fmt.Errorf("parse %s:%d: empty key", absolutePath, lineNumber)
		}
		if lockedKeys[key] {
			continue
		}

		value = strings.TrimSpace(value)
		if unquoted, ok := trimQuoted(value); ok {
			value = unquoted
		}

		if err := os.Setenv(key, value); err != nil {
			return fmt.Errorf("set %s from %s:%d: %w", key, absolutePath, lineNumber, err)
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("scan %s: %w", absolutePath, err)
	}
	return nil
}

func trimQuoted(value string) (string, bool) {
	if len(value) < 2 {
		return "", false
	}

	if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
		unquoted, err := strconv.Unquote(value)
		if err == nil {
			return unquoted, true
		}
		return strings.Trim(value, `"`), true
	}
	if strings.HasPrefix(value, `'`) && strings.HasSuffix(value, `'`) {
		return strings.Trim(value, `'`), true
	}
	return "", false
}

func getenv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getint(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func getfloat(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func getduration(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return parsed
}
