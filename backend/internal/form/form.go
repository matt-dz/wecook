package form

import (
	"errors"
	"fmt"
	"io"
	"net/http"
)

const (
	magicNumberSeek = 512
)

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

	contentType := http.DetectContentType(data[:min(len(data), magicNumberSeek)])
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
