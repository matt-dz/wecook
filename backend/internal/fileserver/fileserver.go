// Package fileserver contains utilities for interacting with the fileserver.
package fileserver

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	directoryPerms = 0o755
	newFilePerms   = 0o644
)

var (
	ErrNotExist    = os.ErrNotExist
	ErrInvalidPath = errors.New("invalid path")
)

// FileServerInterface defines the operations for file server management.
type FileServerInterface interface {
	Delete(path string) error
	Write(path string, data []byte) (fullpath string, n int, err error)
	BaseDirectory() string
}

type FileServer struct {
	baseDirectory string
}

var _ FileServerInterface = (*FileServer)(nil)

func New(baseDirectory string) *FileServer {
	return &FileServer{
		baseDirectory: filepath.Clean(baseDirectory),
	}
}

func (f *FileServer) Write(path string, data []byte) (fullpath string, n int, err error) {
	if f == nil {
		return "", 0, nil
	}

	// Clean path
	fullpath, err = cleanPath(f.baseDirectory, path)
	if err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(filepath.Dir(fullpath), directoryPerms); err != nil {
		return "", 0, fmt.Errorf("creating parent directories: %w", err)
	}

	if info, err := os.Stat(fullpath); err == nil && info.IsDir() {
		return "", 0, errors.Join(fmt.Errorf("path %q is a directory", path), ErrInvalidPath)
	}

	file, err := os.OpenFile(fullpath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, newFilePerms)
	if err != nil {
		return "", 0, fmt.Errorf("creating file: %w", err)
	}
	defer func() { _ = file.Close() }()

	n, err = file.Write(data)
	if err != nil {
		return "", 0, fmt.Errorf("writing file: %w", err)
	}

	return fullpath, n, nil
}

func (f *FileServer) Delete(path string) error {
	if f == nil {
		return nil
	}

	// Clean path
	full, err := cleanPath(f.baseDirectory, path)
	if err != nil {
		return err
	}

	// Ensure it is not the base directory
	base := filepath.Clean(f.baseDirectory)
	if full == base {
		return errors.Join(fmt.Errorf("cannot remove path %q", path), ErrInvalidPath)
	}

	exists := func(path string) (bool, error) {
		_, err := os.Stat(path)
		if err == nil {
			return true, nil
		} else if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}

		return false, err
	}

	// Check existence
	found, err := exists(full)
	if err != nil {
		return fmt.Errorf("checking for existence: %w", err)
	}
	if !found {
		return ErrNotExist
	}

	// Delete file
	if err := os.Remove(full); err != nil {
		return fmt.Errorf("removing file: %w", err)
	}

	// Prune empty directories
	dir := filepath.Dir(full)
	for dir != base && dir != "." && dir != string(filepath.Separator) {
		empty, err := isEmptyDirectory(dir)
		if err != nil {
			return fmt.Errorf("checking empty directory: %w", err)
		}
		if !empty {
			break
		}

		if err := os.Remove(dir); err != nil {
			return fmt.Errorf("removing empty directory: %w", err)
		}
		dir = filepath.Dir(dir)
	}

	return nil
}

func (f *FileServer) BaseDirectory() string {
	if f == nil {
		return ""
	}
	return f.baseDirectory
}

func cleanPath(baseDir, path string) (string, error) {
	cleaned := filepath.Clean(path)
	if strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", ErrInvalidPath
	}

	full := filepath.Join(baseDir, cleaned)
	base, err := filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("getting absolute representation of base directory: %w", err)
	}
	fullAbs, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("getting absolute representation of given directory: %w", err)
	}
	if !strings.HasPrefix(fullAbs, base+string(filepath.Separator)) {
		return "", errors.Join(fmt.Errorf("path escapes base directory"), ErrInvalidPath)
	}

	return fullAbs, nil
}

func topLevelDirectory(path string) string {
	path = strings.TrimPrefix(filepath.Clean(path), string(filepath.Separator))
	tld, _, _ := strings.Cut(path, string(filepath.Separator))
	return tld
}

func isEmptyDirectory(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer func() { _ = f.Close() }()

	// Read one entry — if we get none, it's empty
	_, err = f.Readdir(1)
	if errors.Is(err, io.EOF) {
		return true, nil
	}
	return false, err
}
