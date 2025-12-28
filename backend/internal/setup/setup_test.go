package setup

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/matt-dz/wecook/internal/env"
)

func TestAppSecret_EnvironmentVariableSet(t *testing.T) {
	// Setup
	const secret = "secret"
	secretPath := filepath.Join(t.TempDir(), "secret")
	t.Setenv("APP_SECRET_PATH", secretPath)
	t.Setenv("APP_SECRET", secret)
	env := env.New(nil)
	if env.Get("APP_SECRET_PATH") != secretPath {
		t.Fatalf("APP_SECRET_PATH = %q, want %q", env.Get("APP_SECRET_PATH"), secretPath)
	}
	if env.Get("APP_SECRET") != secret {
		t.Fatalf("failed to set APP_SECRET = %q, want %q", env.Get("APP_SECRET"), secret)
	}

	err := AppSecret(env)
	if err != nil {
		t.Fatalf("AppSecret() received error: %v", err)
	}

	// Secret file should NOT exist
	_, err = os.Lstat(secretPath)
	if err == nil {
		t.Errorf("secret should not be saved - written to %q", secretPath)
	}

	// Expect secret
	if env.Get("APP_SECRET") != secret {
		t.Errorf("APP_SECRET = %q, want %q", env.Get("APP_SECRET"), secret)
	}
}

func TestAppSecret_NoEnvironmentVariable(t *testing.T) {
	// Setup
	const secretFileName = "secret"
	secretPath := filepath.Join(t.TempDir(), secretFileName)
	t.Setenv("APP_SECRET_PATH", secretPath)
	env := env.New(nil)
	if env.Get("APP_SECRET_PATH") != secretPath {
		t.Fatalf("APP_SECRET_PATH = %q, want %q", env.Get("APP_SECRET_PATH"), secretPath)
	}

	err := AppSecret(env)
	if err != nil {
		t.Fatalf("received error: %v", err)
	}

	// Secret file should exist
	f, err := os.Lstat(secretPath)
	if err != nil {
		t.Fatalf("secret file not written")
	}
	if f.Name() != secretFileName {
		t.Fatalf("secret file name = %q, want %q", f.Name(), secretFileName)
	}

	data, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("unexpected error when reading secret file: %v", err)
	}
	if len(data) == 0 {
		t.Errorf("secret file is empty")
	}
	if env.Get("APP_SECRET") == "" {
		t.Errorf("APP_SECRET is empty")
	}
}
