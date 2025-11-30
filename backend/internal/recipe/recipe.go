// Package recipe contains utilities for managing recipes.
package recipe

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	magicNumberSeek = 512
)

type UploadedFile struct {
	Size     int64
	Data     []byte
	Suffix   string
	MimeType string
}

// allowedImageTypes lists the simple MIME types we accept.
var allowedImageTypes = map[string]bool{
	"image/jpeg":    true,
	"image/png":     true,
	"image/svg+xml": true,
	"image/webp":    true,
	"image/gif":     true,
}

var mimeTypeSuffix = map[string]string{
	"image/jpeg":    ".jpg",
	"image/png":     ".png",
	"image/svg+xml": ".svg",
	"image/webp":    ".webp",
	"image/gif":     ".gif",
}

var (
	ErrUnsupportedMimeType = errors.New("unsupported mime type")
	ErrNoImageUploaded     = errors.New("image not uploaded")
)

func ReadImage(r *http.Request, field string) (*UploadedFile, error) {
	f, _, err := r.FormFile(field)
	if errors.Is(err, http.ErrMissingFile) {
		return nil, errors.Join(ErrNoImageUploaded, err)
	} else if err != nil {
		return nil, fmt.Errorf("getting file from form: %w", err)
	}
	defer func() { _ = f.Close() }()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	contentType := http.DetectContentType(data[:min(len(data), magicNumberSeek)])
	if !allowedImageTypes[contentType] {
		return nil, fmt.Errorf("mime type %q: %w", contentType, ErrUnsupportedMimeType)
	}

	return &UploadedFile{
		Size:     int64(len(data)),
		MimeType: contentType,
		Suffix:   mimeTypeSuffix[contentType],
		Data:     data,
	}, nil
}
