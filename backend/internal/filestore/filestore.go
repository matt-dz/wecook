// Package filestore wraps the fileserver package with a more user-friendly interface.
package filestore

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/matt-dz/wecook/internal/fileserver"
)

const (
	ingredientsDir = "ingredients"
	stepsDir       = "steps"
	coverDir       = "covers"
)

const keyIDBytes = 22 // allows for 10^18 ids before likelihood of collision

const (
	KeyPrefix = "/files"
)

type FileStoreInterface interface {
	WriteRecipeCoverImage(suffix string, data []byte) (key string, n int, err error)
	WriteIngredientImage(suffix string, data []byte) (key string, n int, err error)
	WriteStepImage(suffix string, data []byte) (key string, n int, err error)

	DeleteKey(key string) error

	FileURL(key string) string
}

type FileStore struct {
	keyPrefix string
	host      string
	fs        fileserver.FileServerInterface
}

var _ FileStoreInterface = (FileStoreInterface)(FileStore{})

func New(baseDirectory, keyPrefix, host string) FileStore {
	return FileStore{
		keyPrefix: keyPrefix,
		host:      strings.TrimRight(host, "/"),
		fs:        fileserver.New(baseDirectory),
	}
}

func (f FileStore) WriteRecipeCoverImage(suffix string, data []byte) (
	key string, n int, err error,
) {
	// Generate key
	id, err := generateKeyID()
	if err != nil {
		return key, 0, fmt.Errorf("generating key id: %w", err)
	}
	key = coverImageKey(id, suffix)

	// write image
	_, n, err = f.fs.Write(extractKeyPrefix(key, KeyPrefix), data)
	if err != nil {
		return "", n, err
	}
	return key, n, err
}

func (f FileStore) WriteIngredientImage(suffix string, data []byte) (
	key string, n int, err error,
) {
	// Generate key
	id, err := generateKeyID()
	if err != nil {
		return key, 0, fmt.Errorf("generating key id: %w", err)
	}
	key = ingredientsImageKey(id, suffix)

	// write image
	_, n, err = f.fs.Write(extractKeyPrefix(key, KeyPrefix), data)
	if err != nil {
		return "", n, err
	}

	return key, n, err
}

func (f FileStore) WriteStepImage(suffix string, data []byte) (key string, n int, err error) {
	// Generate key
	id, err := generateKeyID()
	if err != nil {
		return key, 0, fmt.Errorf("generating key id: %w", err)
	}
	key = ingredientsStepKey(id, suffix)

	// write key
	_, n, err = f.fs.Write(extractKeyPrefix(key, KeyPrefix), data)
	if err != nil {
		return "", n, err
	}

	return key, n, err
}

func (f FileStore) FileURL(key string) string {
	return f.host + "/" + strings.TrimLeft(key, "/")
}

func (f FileStore) DeleteKey(key string) error {
	return f.fs.Delete(extractKeyPrefix(key, f.keyPrefix))
}

func coverImageKey(id, suffix string) string {
	return filepath.Join(KeyPrefix, coverDir, fmt.Sprintf("%s%s", id, suffix))
}

func ingredientsImageKey(id, suffix string) string {
	return filepath.Join(KeyPrefix, ingredientsDir, fmt.Sprintf("%s%s", id, suffix))
}

func ingredientsStepKey(id, suffix string) string {
	return filepath.Join(KeyPrefix, stepsDir, fmt.Sprintf("%s%s", id, suffix))
}

// extractKeyPrefix removes a leading prefix from a slash-delimited key and
// returns the remaining path with normalized slashes.
//
// Leading and trailing slashes on both key and prefix are ignored when matching.
// i.e. key="files/covers/1.png", prefix="files/" => "covers/1.png".
func extractKeyPrefix(key string, prefix string) string {
	path := strings.Trim(key, "/")
	pathPrefix := strings.Trim(prefix, "/")
	path = strings.TrimPrefix(path, pathPrefix)
	return strings.TrimLeft(path, "/")
}

func generateKeyID() (id string, err error) {
	bytes := make([]byte, keyIDBytes)
	if _, err := rand.Read(bytes); err != nil {
		return id, err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}
