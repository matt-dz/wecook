// Package form contains utilities for handling with http forms.
package form

import (
	"errors"
	"fmt"
	"io"

	"github.com/gabriel-vasile/mimetype"
)

const (
	MaximumUploadSize = 20 << 20 // ~ 20 MB
)

// allowedImageTypes lists the simple MIME types we accept.
var allowedImageTypes = map[string]bool{
	"image/jpeg":    true,
	"image/png":     true,
	"image/svg+xml": true,
	"image/webp":    true,
	"image/gif":     true,
	"image/avif":    true,
	"image/heic":    true, // iPhone default format
	"image/heif":    true, // HEIF variant
	"image/bmp":     true,
	"image/tiff":    true,
}

var mimeTypeSuffix = map[string]string{
	"image/jpeg":    ".jpg",
	"image/png":     ".png",
	"image/svg+xml": ".svg",
	"image/webp":    ".webp",
	"image/gif":     ".gif",
	"image/avif":    ".avif",
	"image/heic":    ".heic",
	"image/heif":    ".heif",
	"image/bmp":     ".bmp",
	"image/tiff":    ".tiff",
}

var (
	ErrUnsupportedMimeType = errors.New("unsupported mime type")
	ErrNoImageUploaded     = errors.New("image not uploaded")
)

type File struct {
	Size     int64
	Data     []byte
	Suffix   string
	MimeType string
}

func ReadFile(file io.ReadCloser) (*File, error) {
	data, err := io.ReadAll(file)
	defer func() { _ = file.Close() }()
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	// Use mimetype package for better detection (supports AVIF and other modern formats)
	mtype := mimetype.Detect(data)
	contentType := mtype.String()

	if !allowedImageTypes[contentType] {
		return nil, fmt.Errorf("mime type %q: %w", contentType, ErrUnsupportedMimeType)
	}

	return &File{
		Size:     int64(len(data)),
		MimeType: contentType,
		Suffix:   mimeTypeSuffix[contentType],
		Data:     data,
	}, nil
}
