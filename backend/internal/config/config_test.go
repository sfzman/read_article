package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvFiles(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	backendDir := filepath.Join(projectDir, "backend")

	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("mkdir backend dir: %v", err)
	}

	rootEnv := "INFERENCE_URL=http://root.example\nDEFAULT_GAP_SECONDS=0.2\n"
	backendEnv := "INFERENCE_URL=http://backend.example\nSERVER_PORT=9090\n"
	if err := os.WriteFile(filepath.Join(projectDir, ".env"), []byte(rootEnv), 0o644); err != nil {
		t.Fatalf("write root .env: %v", err)
	}
	if err := os.WriteFile(filepath.Join(backendDir, ".env"), []byte(backendEnv), 0o644); err != nil {
		t.Fatalf("write backend .env: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(backendDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	t.Setenv("DEFAULT_GAP_SECONDS", "0.3")
	originalInferenceURL, hadInferenceURL := os.LookupEnv("INFERENCE_URL")
	originalServerPort, hadServerPort := os.LookupEnv("SERVER_PORT")
	_ = os.Unsetenv("INFERENCE_URL")
	_ = os.Unsetenv("SERVER_PORT")
	t.Cleanup(func() {
		if hadInferenceURL {
			_ = os.Setenv("INFERENCE_URL", originalInferenceURL)
		} else {
			_ = os.Unsetenv("INFERENCE_URL")
		}
		if hadServerPort {
			_ = os.Setenv("SERVER_PORT", originalServerPort)
		} else {
			_ = os.Unsetenv("SERVER_PORT")
		}
	})

	if err := loadDotEnvFiles(); err != nil {
		t.Fatalf("loadDotEnvFiles() error = %v", err)
	}

	if got := os.Getenv("INFERENCE_URL"); got != "http://backend.example" {
		t.Fatalf("INFERENCE_URL = %q, want backend override", got)
	}
	if got := os.Getenv("SERVER_PORT"); got != "9090" {
		t.Fatalf("SERVER_PORT = %q, want 9090", got)
	}
	if got := os.Getenv("DEFAULT_GAP_SECONDS"); got != "0.3" {
		t.Fatalf("DEFAULT_GAP_SECONDS = %q, want existing env to win", got)
	}
}

func TestLoadResolvesDefaults(t *testing.T) {
	tempDir := t.TempDir()
	projectDir := filepath.Join(tempDir, "project")
	backendDir := filepath.Join(projectDir, "backend")

	if err := os.MkdirAll(backendDir, 0o755); err != nil {
		t.Fatalf("mkdir backend dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".env"), []byte("INFERENCE_URL=http://example\n"), 0o644); err != nil {
		t.Fatalf("write .env: %v", err)
	}

	originalWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(backendDir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(originalWD)
	})

	for _, key := range []string{"SERVER_PORT"} {
		originalValue, hadValue := os.LookupEnv(key)
		_ = os.Unsetenv(key)
		key := key
		t.Cleanup(func() {
			if hadValue {
				_ = os.Setenv(key, originalValue)
			} else {
				_ = os.Unsetenv(key)
			}
		})
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if got := cfg.ServerPort; got != "18080" {
		t.Fatalf("ServerPort = %q, want default 18080", got)
	}
}

func TestTrimQuotedUnescapesDoubleQuotedValues(t *testing.T) {
	t.Parallel()

	got, ok := trimQuoted(`"line1\nline2"`)
	if !ok {
		t.Fatal("trimQuoted() should detect quoted value")
	}
	if got != "line1\nline2" {
		t.Fatalf("trimQuoted() = %q, want escaped newlines to be restored", got)
	}
}
