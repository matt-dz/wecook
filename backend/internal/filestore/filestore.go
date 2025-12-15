// Package filestore wraps the fileserver package with a more user-friendly interface.
package filestore

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/matt-dz/wecook/internal/fileserver"
)

const (
	ingredientsDir = "ingredients"
	stepsDir       = "steps"
	coverDir       = "covers"
)

const (
	DefaultURLPrefix = "/files"
)

type FileStoreInterface interface {
	WriteRecipeCoverImage(recipeID int64, suffix string, data []byte) (urlPath string, n int, err error)
	WriteIngredientImage(recipeID, ingredientID int64, suffix string, data []byte) (urlPath string, n int, err error)
	WriteStepImage(recipeID, stepID int64, suffix string, data []byte) (urlPath string, n int, err error)

	DeleteURLPath(urlpath string) error

	FileURL(host string) string
}

type FileStore struct {
	urlPathPrefix string
	host          string
	fs            fileserver.FileServerInterface
}

var _ FileStoreInterface = (FileStore)(FileStore{})

func New(baseDirectory, urlPathPrefix, host string) FileStore {
	return FileStore{
		urlPathPrefix: urlPathPrefix,
		host:          strings.TrimRight(host, "/"),
		fs:            fileserver.New(baseDirectory),
	}
}

func (f FileStore) WriteRecipeCoverImage(recipeID int64, suffix string, data []byte) (urlPath string, n int, err error) {
	path := coverImagePath(recipeID, suffix)
	fullpath, n, err := f.fs.Write(path, data)
	if err != nil {
		return fullpath, n, err
	}
	return absPathToURLPath(fullpath, f.fs.BaseDirectory(), f.urlPathPrefix), n, err
}

func (f FileStore) WriteIngredientImage(recipeID, ingredientID int64, suffix string, data []byte) (urlPath string, n int, err error) {
	path := ingredientsImagePath(recipeID, ingredientID, suffix)
	fullpath, n, err := f.fs.Write(path, data)
	if err != nil {
		return fullpath, n, err
	}
	return absPathToURLPath(fullpath, f.fs.BaseDirectory(), f.urlPathPrefix), n, err
}

func (f FileStore) WriteStepImage(recipeID, stepID int64, suffix string, data []byte) (urlPath string, n int, err error) {
	path := stepsImagePath(recipeID, stepID, suffix)
	fullpath, n, err := f.fs.Write(path, data)
	if err != nil {
		return fullpath, n, err
	}
	return absPathToURLPath(fullpath, f.fs.BaseDirectory(), f.urlPathPrefix), n, err
}

func (f FileStore) FileURL(urlpath string) string {
	return f.host + "/" + strings.TrimLeft(urlpath, "/")
}

func (f FileStore) DeleteURLPath(urlpath string) error {
	return f.fs.Delete(trimURLPathPrefix(urlpath, f.urlPathPrefix))
}

func stepsImagePath(recipeID, stepID int64, suffix string) string {
	return filepath.Join(stepsDir,
		strconv.FormatInt(recipeID, 10), fmt.Sprintf("%d%s", stepID, suffix))
}

func coverImagePath(recipeID int64, suffix string) string {
	return filepath.Join(coverDir, fmt.Sprintf("%d%s", recipeID, suffix))
}

func ingredientsImagePath(recipeID, ingredientID int64, suffix string) string {
	return filepath.Join(ingredientsDir,
		strconv.FormatInt(recipeID, 10), fmt.Sprintf("%d%s", ingredientID, suffix))
}

func absPathToURLPath(fullpath string, baseDir string, prefix string) (urlpath string) {
	pathPrefix := strings.Trim(prefix, "/")
	relPath := strings.TrimLeft(trimBaseDir(fullpath, baseDir), "/")
	return pathPrefix + "/" + relPath
}

func trimBaseDir(path string, baseDir string) string {
	path = filepath.Clean(path)
	baseDir = filepath.Clean(baseDir)
	return strings.TrimPrefix(path, baseDir)
}

func trimURLPathPrefix(path string, prefix string) string {
	urlpath := strings.Trim(path, "/")
	pathPrefix := strings.Trim(prefix, "/")
	urlpath = strings.TrimPrefix(urlpath, pathPrefix)
	return strings.TrimLeft(urlpath, "/")
}
