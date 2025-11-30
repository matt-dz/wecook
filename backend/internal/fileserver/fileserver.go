// Package fileserver contains utilities for interacting with the fileserver.
package fileserver

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const (
	directoryPerms = 0o755
	urlBaseDir     = "files"
)

const (
	ingredientsDir = "ingredients"
	stepsDir       = "steps"
	coverDir       = "covers"
)

var topLevelDirectories = []string{ingredientsDir, stepsDir, coverDir}

var (
	ErrNotExist    = os.ErrNotExist
	ErrInvalidPath = errors.New("invalid path")
)

type FileServer struct {
	baseDir   string
	serverURL string
}

func New(baseDir, serverURL string) *FileServer {
	return &FileServer{
		baseDir:   baseDir,
		serverURL: serverURL,
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
	if err == nil {
		return true, nil
	} else if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, err
}

func (f *FileServer) Delete(path string) error {
	if f == nil {
		return nil
	}

	// "files/steps/1/1.jpg" -> "steps/1/1.jpg"
	path = filepath.Clean(path)
	path = strings.TrimPrefix(path, string(filepath.Separator))
	path = strings.TrimPrefix(path, urlBaseDir)
	path = strings.TrimPrefix(path, string(filepath.Separator))

	// Clean path
	full, err := cleanPath(f.baseDir, path)
	if err != nil {
		return err
	}
	tld := topLevelDirectory(strings.TrimPrefix(full, f.baseDir))
	if !slices.Contains(topLevelDirectories, tld) {
		return errors.Join(fmt.Errorf("invalid top level directory %q", tld), ErrInvalidPath)
	}

	// Check existence
	found, err := f.Exists(full)
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
	for dir != filepath.Join(f.baseDir, tld) &&
		dir != f.baseDir && dir != "." && dir != string(filepath.Separator) {
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

func (f *FileServer) FileURL(path string) string {
	if f == nil {
		return ""
	}
	var scheme string
	if strings.HasPrefix(f.serverURL, "http://") {
		scheme = "http://"
	} else if strings.HasPrefix(f.serverURL, "https://") {
		scheme = "https://"
	}
	host, _ := strings.CutPrefix(f.serverURL, scheme)
	return scheme + filepath.Join(host, path)
}

func NewStepsImage(recipeID, stepID, suffix string) string {
	return filepath.Join(stepsDir, recipeID, fmt.Sprintf("%s%s", stepID, suffix))
}

func NewCoverImage(recipeID, suffix string) string {
	return filepath.Join(coverDir, fmt.Sprintf("%s%s", recipeID, suffix))
}

func NewIngredientsImage(recipeID, ingredientsID, suffix string) string {
	return filepath.Join(ingredientsDir, recipeID, fmt.Sprintf("%s%s", ingredientsID, suffix))
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
	if !strings.HasPrefix(fullAbs, base+string(filepath.Separator)) &&
		fullAbs != base {
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
