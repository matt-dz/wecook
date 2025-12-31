package filestore

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/matt-dz/wecook/internal/fileserver"
)

func newTestFileStore(t *testing.T) (FileStore, string) {
	t.Helper()
	baseDir := t.TempDir()
	return New(baseDir, KeyPrefix, "http://localhost:8080"), baseDir
}

func TestNew(t *testing.T) {
	baseDir := t.TempDir()
	urlPrefix := "/files"
	host := "http://localhost:8080"

	store := New(baseDir, urlPrefix, host)

	if store.keyPrefix != urlPrefix {
		t.Errorf("urlPathPrefix = %q, want %q", store.keyPrefix, urlPrefix)
	}
	if store.host != host {
		t.Errorf("host = %q, want %q", store.host, host)
	}
	if store.fs == nil {
		t.Error("fs is nil, expected fileserver instance")
	}
}

func TestNew_HostWithTrailingSlash(t *testing.T) {
	baseDir := t.TempDir()
	host := "http://localhost:8080/"

	store := New(baseDir, "/files", host)

	expected := "http://localhost:8080"
	if store.host != expected {
		t.Errorf("host = %q, want %q (trailing slash should be trimmed)", store.host, expected)
	}
}

func TestWriteRecipeCoverImage(t *testing.T) {
	store, baseDir := newTestFileStore(t)
	data := []byte("test cover image data")
	suffix := ".jpg"

	key, n, err := store.WriteRecipeCoverImage(suffix, data)
	if err != nil {
		t.Fatalf("WriteRecipeCoverImage() error = %v", err)
	}

	if n != len(data) {
		t.Errorf("WriteRecipeCoverImage() n = %d, want %d", n, len(data))
	}

	// Verify key format: /files/covers/<random-id>.jpg
	expectedPrefix := filepath.Join(KeyPrefix, coverDir)
	if !strings.HasPrefix(key, expectedPrefix) {
		t.Errorf("WriteRecipeCoverImage() key = %q, should start with %q", key, expectedPrefix)
	}
	if !strings.HasSuffix(key, suffix) {
		t.Errorf("WriteRecipeCoverImage() key = %q, should end with %q", key, suffix)
	}

	// Verify file exists on disk
	relPath := extractKeyPrefix(key, store.keyPrefix)
	expectedFilePath := filepath.Join(baseDir, relPath)
	content, err := os.ReadFile(expectedFilePath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("file content = %q, want %q", string(content), string(data))
	}
}

func TestWriteIngredientImage(t *testing.T) {
	store, baseDir := newTestFileStore(t)
	data := []byte("test ingredient image")
	suffix := ".png"

	key, n, err := store.WriteIngredientImage(suffix, data)
	if err != nil {
		t.Fatalf("WriteIngredientImage() error = %v", err)
	}

	if n != len(data) {
		t.Errorf("WriteIngredientImage() n = %d, want %d", n, len(data))
	}

	// Verify key format: /files/ingredients/<random-id>.png
	expectedPrefix := filepath.Join(KeyPrefix, ingredientsDir)
	if !strings.HasPrefix(key, expectedPrefix) {
		t.Errorf("WriteIngredientImage() key = %q, should start with %q", key, expectedPrefix)
	}
	if !strings.HasSuffix(key, suffix) {
		t.Errorf("WriteIngredientImage() key = %q, should end with %q", key, suffix)
	}

	// Verify file exists on disk
	relPath := extractKeyPrefix(key, store.keyPrefix)
	expectedFilePath := filepath.Join(baseDir, relPath)
	content, err := os.ReadFile(expectedFilePath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("file content = %q, want %q", string(content), string(data))
	}
}

func TestWriteStepImage(t *testing.T) {
	store, baseDir := newTestFileStore(t)
	data := []byte("test step image")
	suffix := ".webp"

	key, n, err := store.WriteStepImage(suffix, data)
	if err != nil {
		t.Fatalf("WriteStepImage() error = %v", err)
	}

	if n != len(data) {
		t.Errorf("WriteStepImage() n = %d, want %d", n, len(data))
	}

	// Verify key format: /files/steps/<random-id>.webp
	expectedPrefix := filepath.Join(KeyPrefix, stepsDir)
	if !strings.HasPrefix(key, expectedPrefix) {
		t.Errorf("WriteStepImage() key = %q, should start with %q", key, expectedPrefix)
	}
	if !strings.HasSuffix(key, suffix) {
		t.Errorf("WriteStepImage() key = %q, should end with %q", key, suffix)
	}

	// Verify file exists on disk
	relPath := extractKeyPrefix(key, store.keyPrefix)
	expectedFilePath := filepath.Join(baseDir, relPath)
	content, err := os.ReadFile(expectedFilePath)
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}
	if string(content) != string(data) {
		t.Errorf("file content = %q, want %q", string(content), string(data))
	}
}

func TestFileURL(t *testing.T) {
	tests := []struct {
		name     string
		host     string
		key      string
		expected string
	}{
		{
			name:     "simple key",
			host:     "http://localhost:8080",
			key:      "/files/covers/abc123.jpg",
			expected: "http://localhost:8080/files/covers/abc123.jpg",
		},
		{
			name:     "key without leading slash",
			host:     "http://localhost:8080",
			key:      "files/covers/abc123.jpg",
			expected: "http://localhost:8080/files/covers/abc123.jpg",
		},
		{
			name:     "production host",
			host:     "https://api.example.com",
			key:      "/files/steps/xyz789.png",
			expected: "https://api.example.com/files/steps/xyz789.png",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			store := FileStore{
				host:      tt.host,
				keyPrefix: KeyPrefix,
				fs:        fileserver.New(baseDir),
			}

			got := store.FileURL(tt.key)
			if got != tt.expected {
				t.Errorf("FileURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDeleteKey(t *testing.T) {
	store, baseDir := newTestFileStore(t)

	// First, write a file
	data := []byte("test data")
	key, _, err := store.WriteRecipeCoverImage(".jpg", data)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(baseDir, extractKeyPrefix(key, store.keyPrefix))
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("file should exist before delete: %v", err)
	}

	// Delete using key
	err = store.DeleteKey(key)
	if err != nil {
		t.Fatalf("DeleteKey() error = %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected file to be deleted, got err = %v", err)
	}
}

func TestDeleteKey_NonExistent(t *testing.T) {
	store, _ := newTestFileStore(t)

	err := store.DeleteKey("/files/covers/nonexistent.jpg")
	if !errors.Is(err, fileserver.ErrNotExist) {
		t.Errorf("DeleteKey() error = %v, want ErrNotExist", err)
	}
}

func TestDeleteKey_VariousPrefixes(t *testing.T) {
	tests := []struct {
		name string
		key  string
	}{
		{
			name: "with leading slash",
			key:  "/files/covers/abc123.jpg",
		},
		{
			name: "without leading slash",
			key:  "files/covers/abc123.jpg",
		},
		{
			name: "with trailing slash",
			key:  "/files/covers/abc123.jpg/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, baseDir := newTestFileStore(t)

			// Create test file
			filePath := filepath.Join(baseDir, "covers", "abc123.jpg")
			if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
				t.Fatalf("failed to create directories: %v", err)
			}
			if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Delete
			err := store.DeleteKey(tt.key)
			if err != nil {
				t.Fatalf("DeleteKey() error = %v", err)
			}

			// Verify deletion
			if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("expected file to be deleted")
			}
		})
	}
}

func TestCoverImageKey(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		suffix   string
		expected string
	}{
		{
			name:     "jpg image",
			id:       "abc123",
			suffix:   ".jpg",
			expected: filepath.Join(KeyPrefix, "covers", "abc123.jpg"),
		},
		{
			name:     "png image",
			id:       "xyz789",
			suffix:   ".png",
			expected: filepath.Join(KeyPrefix, "covers", "xyz789.png"),
		},
		{
			name:     "no extension",
			id:       "test",
			suffix:   "",
			expected: filepath.Join(KeyPrefix, "covers", "test"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coverImageKey(tt.id, tt.suffix)
			if got != tt.expected {
				t.Errorf("coverImageKey() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIngredientsImageKey(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		suffix   string
		expected string
	}{
		{
			name:     "basic path",
			id:       "ing200",
			suffix:   ".jpg",
			expected: filepath.Join(KeyPrefix, "ingredients", "ing200.jpg"),
		},
		{
			name:     "another path",
			id:       "abc888",
			suffix:   ".webp",
			expected: filepath.Join(KeyPrefix, "ingredients", "abc888.webp"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ingredientsImageKey(tt.id, tt.suffix)
			if got != tt.expected {
				t.Errorf("ingredientsImageKey() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestExtractKeyPrefix(t *testing.T) {
	tests := []struct {
		name     string
		key      string
		prefix   string
		expected string
	}{
		{
			name:     "trim leading prefix",
			key:      "/files/covers/123.jpg",
			prefix:   "/files",
			expected: "covers/123.jpg",
		},
		{
			name:     "key without leading slash",
			key:      "files/covers/123.jpg",
			prefix:   "/files",
			expected: "covers/123.jpg",
		},
		{
			name:     "prefix without slashes",
			key:      "/static/images/1.jpg",
			prefix:   "static",
			expected: "images/1.jpg",
		},
		{
			name:     "trailing slash in key",
			key:      "/files/covers/123.jpg/",
			prefix:   "/files",
			expected: "covers/123.jpg",
		},
		{
			name:     "both without slashes",
			key:      "api/v1/resource",
			prefix:   "api",
			expected: "v1/resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractKeyPrefix(tt.key, tt.prefix)
			if got != tt.expected {
				t.Errorf("extractKeyPrefix() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestGenerateKeyId(t *testing.T) {
	// Test that generateKeyId produces unique IDs
	seen := make(map[string]bool)
	for range 100 {
		id, err := generateKeyID()
		if err != nil {
			t.Fatalf("generateKeyId() error = %v", err)
		}
		if id == "" {
			t.Error("generateKeyId() returned empty string")
		}
		if seen[id] {
			t.Errorf("generateKeyId() produced duplicate ID: %q", id)
		}
		seen[id] = true

		// Verify it's valid base64 URL encoding
		if strings.ContainsAny(id, "+/=") {
			t.Errorf("generateKeyId() = %q, should use URL-safe base64 encoding", id)
		}
	}
}

func TestIntegration_WriteAndDelete(t *testing.T) {
	store, baseDir := newTestFileStore(t)

	// Write a cover image
	coverData := []byte("cover image data")
	key, _, err := store.WriteRecipeCoverImage(".jpg", coverData)
	if err != nil {
		t.Fatalf("WriteRecipeCoverImage() error = %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(baseDir, extractKeyPrefix(key, store.keyPrefix))
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("file should exist after write: %v", err)
	}

	// Delete using key
	if err := store.DeleteKey(key); err != nil {
		t.Fatalf("DeleteKey() error = %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file should not exist after delete")
	}
}

func TestIntegration_MultipleImages(t *testing.T) {
	store, _ := newTestFileStore(t)

	// Write cover
	coverKey, _, err := store.WriteRecipeCoverImage(".jpg", []byte("cover"))
	if err != nil {
		t.Fatalf("WriteRecipeCoverImage() error = %v", err)
	}

	// Write multiple steps
	step1Key, _, err := store.WriteStepImage(".jpg", []byte("step1"))
	if err != nil {
		t.Fatalf("WriteStepImage(1) error = %v", err)
	}
	step2Key, _, err := store.WriteStepImage(".jpg", []byte("step2"))
	if err != nil {
		t.Fatalf("WriteStepImage(2) error = %v", err)
	}

	// Write multiple ingredients
	ing1Key, _, err := store.WriteIngredientImage(".png", []byte("ing1"))
	if err != nil {
		t.Fatalf("WriteIngredientImage(1) error = %v", err)
	}
	ing2Key, _, err := store.WriteIngredientImage(".png", []byte("ing2"))
	if err != nil {
		t.Fatalf("WriteIngredientImage(2) error = %v", err)
	}

	// Verify all keys are different (they should have unique random IDs)
	keys := []string{coverKey, step1Key, step2Key, ing1Key, ing2Key}
	seen := make(map[string]bool)
	for _, key := range keys {
		if seen[key] {
			t.Errorf("duplicate key found: %q", key)
		}
		seen[key] = true
	}

	// Delete all and verify
	for _, key := range keys {
		if err := store.DeleteKey(key); err != nil {
			t.Errorf("DeleteKey(%q) error = %v", key, err)
		}
	}
}
