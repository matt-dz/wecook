// Package fileserver contains utilities for interacting with the fileserver.
package fileserver

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	directoryPerms = 0o755
	urlBaseDir     = "files"
)

const (
	IngredientsDir = "ingredients"
	StepsDir       = "steps"
)

type FileServer struct {
	baseDir string
}

func New(baseDir string) *FileServer {
	return &FileServer{
		baseDir: baseDir,
	}
}

func (f *FileServer) Write(path string, data []byte) (location string, n int, err error) {
	if f == nil {
		return "", 0, nil
	}

	fullpath := filepath.Join(f.baseDir, path)
	if err := os.MkdirAll(filepath.Dir(fullpath), directoryPerms); err != nil {
		return "", 0, fmt.Errorf("creating parent directories: %w", err)
	}

	file, err := os.Create(fullpath)
	if err != nil {
		return "", 0, fmt.Errorf("creating file: %w", err)
	}
	defer func() { _ = file.Close() }()

	n, err = file.Write(data)
	if err != nil {
		return "", 0, fmt.Errorf("writing file: %w", err)
	}

	return filepath.Join(urlBaseDir, path), n, nil
}

func (f *FileServer) Exists(path string) (bool, error) {
	if f == nil {
		return false, nil
	}
	_, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return true, nil
}
