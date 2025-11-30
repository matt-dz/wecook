package fileserver

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// helper to create a test FileServer rooted at a temp dir.
func newTestFileServer(t *testing.T) (*FileServer, string) {
	t.Helper()

	base := t.TempDir()

	// Make sure allowed top-level dirs are predictable for tests
	topLevelDirectories = []string{"covers", "steps"}

	// Minimal construction: assumes FileServer has a baseDir field
	return &FileServer{baseDir: base}, base
}

func TestCleanPath_Valid(t *testing.T) {
	baseDir := filepath.Join("testdata", "base")

	tests := []struct {
		name     string
		path     string
		expected string // expected cleaned relative part under base
	}{
		{
			name:     "simple relative path",
			path:     "images/foo.png",
			expected: filepath.Join("images", "foo.png"),
		},
		{
			name:     "path with dot segments",
			path:     "./images/./foo.png",
			expected: filepath.Join("images", "foo.png"),
		},
		{
			name:     "path with inner dot-dot but still inside",
			path:     "images/2025/../foo.png",
			expected: filepath.Join("images", "foo.png"),
		},
		{
			name:     "empty path resolves to base",
			path:     "",
			expected: ".",
		},
		{
			name:     "dot path resolves to base",
			path:     ".",
			expected: ".",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := cleanPath(baseDir, tt.path)
			if err != nil {
				t.Fatalf("cleanPath() returned unexpected error: %v", err)
			}

			absBase, err := filepath.Abs(baseDir)
			if err != nil {
				t.Fatalf("failed to get abs base: %v", err)
			}

			// expected absolute full path
			want := filepath.Join(absBase, tt.expected)
			want, err = filepath.Abs(want)
			if err != nil {
				t.Fatalf("failed to get abs expected: %v", err)
			}

			if got != want {
				t.Fatalf("cleanPath() = %q, want %q", got, want)
			}
		})
	}
}

func TestCleanPath_Invalid(t *testing.T) {
	baseDir := filepath.Join("testdata", "base")

	absoluteBad := filepath.Join(string(filepath.Separator), "etc", "passwd")

	tests := []struct {
		name string
		path string
	}{
		{
			name: "starts with dot-dot",
			path: "../secret.txt",
		},
		{
			name: "cleaned becomes dot-dot",
			path: "foo/../../secret.txt",
		},
		{
			name: "absolute path outside base",
			path: absoluteBad,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := cleanPath(baseDir, tt.path)
			if err == nil {
				t.Fatalf("cleanPath(%q) = %q, expected error", tt.path, got)
			}
			if !errors.Is(err, ErrInvalidPath) {
				t.Fatalf("cleanPath(%q) error = %v, want ErrInvalidPath", tt.path, err)
			}
		})
	}
}

func TestCleanPath_DoesNotEscapeBase(t *testing.T) {
	// baseDir like /tmp/base
	baseDir := filepath.Join("testdata", "base")

	// A sneaky path that tries to escape *after* base join
	// e.g. baseDir = /data/base, path = "../outside" should be rejected.
	path := "../outside.txt"

	got, err := cleanPath(baseDir, path)
	if err == nil {
		t.Fatalf("expected error for escaping path, got %q", got)
	}
	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected ErrInvalidPath, got %v", err)
	}
}

func TestCleanPath_AllowsBaseItself(t *testing.T) {
	baseDir := filepath.Join("testdata", "base")

	got, err := cleanPath(baseDir, ".")
	if err != nil {
		t.Fatalf("cleanPath() returned unexpected error: %v", err)
	}

	absBase, err := filepath.Abs(baseDir)
	if err != nil {
		t.Fatalf("failed to get abs base: %v", err)
	}

	if got != absBase {
		t.Fatalf("cleanPath() for '.' = %q, want %q", got, absBase)
	}
}

func TestTopLevelDirectory(t *testing.T) {
	sep := string(filepath.Separator)

	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "images/foo/bar.png",
			expected: "images",
		},
		{
			name:     "path with leading slash",
			path:     sep + "images/foo/bar.png",
			expected: "images",
		},
		{
			name:     "path with dot segments",
			path:     "./images/./foo",
			expected: "images",
		},
		{
			name:     "path with inner .. that still stays inside",
			path:     "a/b/../c/d",
			expected: "a",
		},
		{
			name:     "single top-level directory",
			path:     "images",
			expected: "images",
		},
		{
			name:     "single top-level directory with slash",
			path:     "images" + sep,
			expected: "images",
		},
		{
			name:     "empty path becomes empty",
			path:     "",
			expected: ".",
		},
		{
			name:     "root-only slash",
			path:     sep,
			expected: "",
		},
		{
			name:     "dot path",
			path:     ".",
			expected: ".", // filepath.Clean(".") returns "."
		},
		{
			name:     "double slash collapse",
			path:     sep + sep + "foo/bar",
			expected: "foo",
		},
		{
			name:     "leading directory traversals",
			path:     "../foo",
			expected: "..", // Clean("../foo") = "../foo", TrimPrefix("../foo", "/") = "../foo"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := topLevelDirectory(tt.path)
			if got != tt.expected {
				t.Fatalf("topLevelDirectory(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestIsEmptyDirectory(t *testing.T) {
	t.Run("empty directory", func(t *testing.T) {
		dir := t.TempDir()

		ok, err := isEmptyDirectory(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !ok {
			t.Fatalf("expected empty dir, got not empty")
		}
	})

	t.Run("directory with one file", func(t *testing.T) {
		dir := t.TempDir()

		err := os.WriteFile(filepath.Join(dir, "file.txt"), []byte("hello"), 0o644)
		if err != nil {
			t.Fatalf("failed to write test file: %v", err)
		}

		ok, err := isEmptyDirectory(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Fatalf("expected non-empty dir, got empty")
		}
	})

	t.Run("directory with subdirectory", func(t *testing.T) {
		dir := t.TempDir()

		err := os.Mkdir(filepath.Join(dir, "child"), 0o755)
		if err != nil {
			t.Fatalf("failed to create child dir: %v", err)
		}

		ok, err := isEmptyDirectory(dir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if ok {
			t.Fatalf("expected non-empty dir, got empty")
		}
	})

	t.Run("non-existent directory", func(t *testing.T) {
		_, err := isEmptyDirectory("does/not/exist")
		if err == nil {
			t.Fatalf("expected error for non-existent directory, got nil")
		}
	})

	t.Run("path is a file", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "file.txt")

		err := os.WriteFile(file, []byte("hello"), fs.ModePerm)
		if err != nil {
			t.Fatalf("failed to write file: %v", err)
		}

		// Expect os.Open() on a file to succeed, but Readdir should error
		ok, err := isEmptyDirectory(file)
		if err == nil {
			t.Fatalf("expected error for file path, got ok=%v err=nil", ok)
		}
	})
}

func TestFileServerDelete_SuccessAndPruneEmptyDirs(t *testing.T) {
	fs, base := newTestFileServer(t)

	// Create path: <base>/covers/recipe1/step1.png
	relPath := filepath.Join("covers", "recipe1", "step1.png")
	fullPath := filepath.Join(base, relPath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		t.Fatalf("failed to create dirs: %v", err)
	}
	if err := os.WriteFile(fullPath, []byte("data"), 0o644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}

	// Delete should remove the file and prune "recipe1" dir,
	// but keep the "covers" top-level directory and baseDir.
	if err := fs.Delete(relPath); err != nil {
		t.Fatalf("Delete() returned error: %v", err)
	}

	// File must be gone
	if _, err := os.Stat(fullPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected file to be removed, got err=%v", err)
	}

	// "recipe1" directory should be gone (pruned)
	if _, err := os.Stat(filepath.Join(base, "covers", "recipe1")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected recipe1 directory to be removed, got err=%v", err)
	}

	// "covers" directory should still exist (top-level)
	if _, err := os.Stat(filepath.Join(base, "covers")); err != nil {
		t.Fatalf("expected covers directory to remain, got err=%v", err)
	}

	// base directory should still exist
	if _, err := os.Stat(base); err != nil {
		t.Fatalf("expected base directory to remain, got err=%v", err)
	}
}

func TestFileServerDelete_InvalidTopLevelDir(t *testing.T) {
	fs, _ := newTestFileServer(t)

	// topLevelDirectories = []string{"covers", "steps"} from helper
	// "other" should be rejected
	err := fs.Delete(filepath.Join("other", "file.png"))
	if err == nil {
		t.Fatalf("expected error for invalid top-level directory, got nil")
	}
	if !errors.Is(err, ErrInvalidPath) {
		t.Fatalf("expected ErrInvalidPath, got %v", err)
	}
}

func TestFileServerDelete_FileDoesNotExist(t *testing.T) {
	fs, base := newTestFileServer(t)

	relPath := filepath.Join("covers", "recipe1", "missing.png")
	fullPath := filepath.Join(base, relPath)

	// Ensure it really doesn't exist
	_ = os.Remove(fullPath)

	err := fs.Delete(relPath)
	if !errors.Is(err, ErrNotExist) {
		t.Fatalf("expected ErrNotExist for missing file, got %v", err)
	}
}

func TestFileServerDelete_NilReceiverNoop(t *testing.T) {
	var fs *FileServer

	// Should not panic and should return nil
	if err := fs.Delete("covers/recipe1/step1.png"); err != nil {
		t.Fatalf("expected nil error on nil receiver, got %v", err)
	}
}
