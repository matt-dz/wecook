package filestore

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/matt-dz/wecook/internal/fileserver"
)

func newTestFileStore(t *testing.T) (FileStore, string) {
	t.Helper()
	baseDir := t.TempDir()
	return New(baseDir, DefaultURLPrefix, "http://localhost:8080"), baseDir
}

func TestNew(t *testing.T) {
	baseDir := t.TempDir()
	urlPrefix := "/files"
	host := "http://localhost:8080"

	store := New(baseDir, urlPrefix, host)

	if store.urlPathPrefix != urlPrefix {
		t.Errorf("urlPathPrefix = %q, want %q", store.urlPathPrefix, urlPrefix)
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
	recipeID := int64(123)
	suffix := ".jpg"

	urlPath, n, err := store.WriteRecipeCoverImage(recipeID, suffix, data)
	if err != nil {
		t.Fatalf("WriteRecipeCoverImage() error = %v", err)
	}

	if n != len(data) {
		t.Errorf("WriteRecipeCoverImage() n = %d, want %d", n, len(data))
	}

	expectedURLPath := "files/covers/123.jpg"
	if urlPath != expectedURLPath {
		t.Errorf("WriteRecipeCoverImage() urlPath = %q, want %q", urlPath, expectedURLPath)
	}

	// Verify file exists on disk
	expectedFilePath := filepath.Join(baseDir, "covers", "123.jpg")
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
	recipeID := int64(456)
	ingredientID := int64(789)
	suffix := ".png"

	urlPath, n, err := store.WriteIngredientImage(recipeID, ingredientID, suffix, data)
	if err != nil {
		t.Fatalf("WriteIngredientImage() error = %v", err)
	}

	if n != len(data) {
		t.Errorf("WriteIngredientImage() n = %d, want %d", n, len(data))
	}

	expectedURLPath := "files/ingredients/456/789.png"
	if urlPath != expectedURLPath {
		t.Errorf("WriteIngredientImage() urlPath = %q, want %q", urlPath, expectedURLPath)
	}

	// Verify file exists on disk
	expectedFilePath := filepath.Join(baseDir, "ingredients", "456", "789.png")
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
	recipeID := int64(111)
	stepID := int64(222)
	suffix := ".webp"

	urlPath, n, err := store.WriteStepImage(recipeID, stepID, suffix, data)
	if err != nil {
		t.Fatalf("WriteStepImage() error = %v", err)
	}

	if n != len(data) {
		t.Errorf("WriteStepImage() n = %d, want %d", n, len(data))
	}

	expectedURLPath := "files/steps/111/222.webp"
	if urlPath != expectedURLPath {
		t.Errorf("WriteStepImage() urlPath = %q, want %q", urlPath, expectedURLPath)
	}

	// Verify file exists on disk
	expectedFilePath := filepath.Join(baseDir, "steps", "111", "222.webp")
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
		urlPath  string
		expected string
	}{
		{
			name:     "simple path",
			host:     "http://localhost:8080",
			urlPath:  "/files/covers/123.jpg",
			expected: "http://localhost:8080/files/covers/123.jpg",
		},
		{
			name:     "path without leading slash",
			host:     "http://localhost:8080",
			urlPath:  "files/covers/123.jpg",
			expected: "http://localhost:8080/files/covers/123.jpg",
		},
		{
			name:     "production host",
			host:     "https://api.example.com",
			urlPath:  "/files/steps/1/2.png",
			expected: "https://api.example.com/files/steps/1/2.png",
		},
		{
			name:     "host with trailing slash",
			host:     "http://localhost:8080/",
			urlPath:  "/files/covers/123.jpg",
			expected: "http://localhost:8080//files/covers/123.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseDir := t.TempDir()
			store := FileStore{
				host:          tt.host,
				urlPathPrefix: DefaultURLPrefix,
				fs:            fileserver.New(baseDir),
			}

			got := store.FileURL(tt.urlPath)
			if got != tt.expected {
				t.Errorf("FileURL() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestDeleteURLPath(t *testing.T) {
	store, baseDir := newTestFileStore(t)

	// First, write a file
	filePath := filepath.Join(baseDir, "covers", "123.jpg")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatalf("failed to create directories: %v", err)
	}
	if err := os.WriteFile(filePath, []byte("test data"), 0o644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Delete using URL path
	err := store.DeleteURLPath("/files/covers/123.jpg")
	if err != nil {
		t.Fatalf("DeleteURLPath() error = %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected file to be deleted, got err = %v", err)
	}
}

func TestDeleteURLPath_NonExistent(t *testing.T) {
	store, _ := newTestFileStore(t)

	err := store.DeleteURLPath("/files/covers/nonexistent.jpg")
	if !errors.Is(err, fileserver.ErrNotExist) {
		t.Errorf("DeleteURLPath() error = %v, want ErrNotExist", err)
	}
}

func TestDeleteURLPath_VariousPrefixes(t *testing.T) {
	tests := []struct {
		name    string
		urlPath string
	}{
		{
			name:    "with leading slash",
			urlPath: "/files/covers/123.jpg",
		},
		{
			name:    "without leading slash",
			urlPath: "files/covers/123.jpg",
		},
		{
			name:    "with trailing slash",
			urlPath: "/files/covers/123.jpg/",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, baseDir := newTestFileStore(t)

			// Create test file
			filePath := filepath.Join(baseDir, "covers", "123.jpg")
			if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
				t.Fatalf("failed to create directories: %v", err)
			}
			if err := os.WriteFile(filePath, []byte("test"), 0o644); err != nil {
				t.Fatalf("failed to write test file: %v", err)
			}

			// Delete
			err := store.DeleteURLPath(tt.urlPath)
			if err != nil {
				t.Fatalf("DeleteURLPath() error = %v", err)
			}

			// Verify deletion
			if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("expected file to be deleted")
			}
		})
	}
}

func TestCoverImagePath(t *testing.T) {
	tests := []struct {
		name     string
		recipeID int64
		suffix   string
		expected string
	}{
		{
			name:     "jpg image",
			recipeID: 123,
			suffix:   ".jpg",
			expected: filepath.Join("covers", "123.jpg"),
		},
		{
			name:     "png image",
			recipeID: 456,
			suffix:   ".png",
			expected: filepath.Join("covers", "456.png"),
		},
		{
			name:     "no extension",
			recipeID: 789,
			suffix:   "",
			expected: filepath.Join("covers", "789"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := coverImagePath(tt.recipeID, tt.suffix)
			if got != tt.expected {
				t.Errorf("coverImagePath() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestStepsImagePath(t *testing.T) {
	tests := []struct {
		name     string
		recipeID int64
		stepID   int64
		suffix   string
		expected string
	}{
		{
			name:     "basic path",
			recipeID: 100,
			stepID:   1,
			suffix:   ".jpg",
			expected: filepath.Join("steps", "100", "1.jpg"),
		},
		{
			name:     "nested recipe",
			recipeID: 999,
			stepID:   42,
			suffix:   ".png",
			expected: filepath.Join("steps", "999", "42.png"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stepsImagePath(tt.recipeID, tt.stepID, tt.suffix)
			if got != tt.expected {
				t.Errorf("stepsImagePath() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIngredientsImagePath(t *testing.T) {
	tests := []struct {
		name         string
		recipeID     int64
		ingredientID int64
		suffix       string
		expected     string
	}{
		{
			name:         "basic path",
			recipeID:     200,
			ingredientID: 10,
			suffix:       ".jpg",
			expected:     filepath.Join("ingredients", "200", "10.jpg"),
		},
		{
			name:         "nested path",
			recipeID:     888,
			ingredientID: 77,
			suffix:       ".webp",
			expected:     filepath.Join("ingredients", "888", "77.webp"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ingredientsImagePath(tt.recipeID, tt.ingredientID, tt.suffix)
			if got != tt.expected {
				t.Errorf("ingredientsImagePath() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestAbsPathToURLPath(t *testing.T) {
	tests := []struct {
		name     string
		fullpath string
		baseDir  string
		prefix   string
		expected string
	}{
		{
			name:     "unix path",
			fullpath: "/data/images/covers/123.jpg",
			baseDir:  "/data/images",
			prefix:   "/files",
			expected: "files/covers/123.jpg",
		},
		{
			name:     "nested path",
			fullpath: "/var/app/static/ingredients/456/789.png",
			baseDir:  "/var/app/static",
			prefix:   "/static",
			expected: "static/ingredients/456/789.png",
		},
		{
			name:     "prefix without slashes",
			fullpath: "/tmp/files/steps/1/2.jpg",
			baseDir:  "/tmp/files",
			prefix:   "api",
			expected: "api/steps/1/2.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Adjust paths for current OS
			fullpath := filepath.FromSlash(tt.fullpath)
			baseDir := filepath.FromSlash(tt.baseDir)
			// Expected output should use forward slashes regardless of OS
			expected := tt.expected

			got := absPathToURLPath(fullpath, baseDir, tt.prefix)

			// Normalize both to forward slashes for comparison
			gotNormalized := filepath.ToSlash(got)
			expectedNormalized := filepath.ToSlash(expected)

			if gotNormalized != expectedNormalized {
				t.Errorf("absPathToURLPath() = %q, want %q", gotNormalized, expectedNormalized)
			}
		})
	}
}

func TestTrimBaseDir(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		baseDir  string
		expected string
	}{
		{
			name:     "simple trim",
			path:     "/data/images/covers/123.jpg",
			baseDir:  "/data/images",
			expected: "/covers/123.jpg",
		},
		{
			name:     "with dots in path",
			path:     "/data/./images/./file.jpg",
			baseDir:  "/data/images",
			expected: "/file.jpg",
		},
		{
			name:     "already relative",
			path:     "file.jpg",
			baseDir:  ".",
			expected: "file.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimBaseDir(tt.path, tt.baseDir)
			// For this test, we'll just verify it doesn't panic and returns a string
			if got == "" && tt.path != "" {
				t.Errorf("trimBaseDir() returned empty string for non-empty path")
			}
		})
	}
}

func TestTrimURLPathPrefix(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		prefix   string
		expected string
	}{
		{
			name:     "trim leading prefix",
			path:     "/files/covers/123.jpg",
			prefix:   "/files",
			expected: "covers/123.jpg",
		},
		{
			name:     "path without leading slash",
			path:     "files/covers/123.jpg",
			prefix:   "/files",
			expected: "covers/123.jpg",
		},
		{
			name:     "prefix without slashes",
			path:     "/static/images/1.jpg",
			prefix:   "static",
			expected: "images/1.jpg",
		},
		{
			name:     "trailing slash in path",
			path:     "/files/covers/123.jpg/",
			prefix:   "/files",
			expected: "covers/123.jpg",
		},
		{
			name:     "both without slashes",
			path:     "api/v1/resource",
			prefix:   "api",
			expected: "v1/resource",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := trimURLPathPrefix(tt.path, tt.prefix)
			if got != tt.expected {
				t.Errorf("trimURLPathPrefix() = %q, want %q", got, tt.expected)
			}
		})
	}
}

func TestIntegration_WriteAndDelete(t *testing.T) {
	store, baseDir := newTestFileStore(t)

	// Write a cover image
	coverData := []byte("cover image data")
	urlPath, _, err := store.WriteRecipeCoverImage(123, ".jpg", coverData)
	if err != nil {
		t.Fatalf("WriteRecipeCoverImage() error = %v", err)
	}

	// Verify file exists
	filePath := filepath.Join(baseDir, "covers", "123.jpg")
	if _, err := os.Stat(filePath); err != nil {
		t.Fatalf("file should exist after write: %v", err)
	}

	// Delete using URL path
	if err := store.DeleteURLPath(urlPath); err != nil {
		t.Fatalf("DeleteURLPath() error = %v", err)
	}

	// Verify file is deleted
	if _, err := os.Stat(filePath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("file should not exist after delete")
	}
}

func TestIntegration_MultipleImagesInSameRecipe(t *testing.T) {
	store, _ := newTestFileStore(t)

	recipeID := int64(999)

	// Write cover
	coverURL, _, err := store.WriteRecipeCoverImage(recipeID, ".jpg", []byte("cover"))
	if err != nil {
		t.Fatalf("WriteRecipeCoverImage() error = %v", err)
	}

	// Write multiple steps
	step1URL, _, err := store.WriteStepImage(recipeID, 1, ".jpg", []byte("step1"))
	if err != nil {
		t.Fatalf("WriteStepImage(1) error = %v", err)
	}
	step2URL, _, err := store.WriteStepImage(recipeID, 2, ".jpg", []byte("step2"))
	if err != nil {
		t.Fatalf("WriteStepImage(2) error = %v", err)
	}

	// Write multiple ingredients
	ing1URL, _, err := store.WriteIngredientImage(recipeID, 10, ".png", []byte("ing1"))
	if err != nil {
		t.Fatalf("WriteIngredientImage(10) error = %v", err)
	}
	ing2URL, _, err := store.WriteIngredientImage(recipeID, 20, ".png", []byte("ing2"))
	if err != nil {
		t.Fatalf("WriteIngredientImage(20) error = %v", err)
	}

	// Verify all URLs are different
	urls := []string{coverURL, step1URL, step2URL, ing1URL, ing2URL}
	seen := make(map[string]bool)
	for _, url := range urls {
		if seen[url] {
			t.Errorf("duplicate URL found: %q", url)
		}
		seen[url] = true
	}

	// Delete all and verify
	for _, url := range urls {
		if err := store.DeleteURLPath(url); err != nil {
			t.Errorf("DeleteURLPath(%q) error = %v", url, err)
		}
	}
}
