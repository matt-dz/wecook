// Package recipe contains utilities for managing recipes.
package recipe

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	magicNumberSeek = 512
)

type RecipeOwner struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

type RecipeIngredient struct {
	ID       int64   `json:"id"`
	RecipeID int64   `json:"recipe_id"`
	Quantity float32 `json:"quantity,omitempty"`
	Name     string  `json:"name"`
	Unit     string  `json:"unit,omitempty"`
	ImageURL string  `json:"image_url,omitempty"`
}

type RecipeStep struct {
	StepNumber  int32     `json:"step_number"`
	ID          int64     `json:"id"`
	RecipeID    int64     `json:"recipe_id"`
	Instruction string    `json:"instruction"`
	ImageURL    string    `json:"image_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Recipe struct {
	Published        bool      `json:"published"`
	CookeTimeMinutes uint32    `json:"cook_time_minutes"`
	Servings         float32   `json:"servings"`
	UserID           int64     `json:"user_id"`
	ID               int64     `json:"id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
	ImageURL         string    `json:"image_url,omitempty"`
	Title            string    `json:"title"`
	Description      string    `json:"description,omitempty"`
}

type RecipeAndOwner struct {
	Recipe Recipe      `json:"recipe"`
	Owner  RecipeOwner `json:"owner"`
}

type RecipeWithIngredientsAndSteps struct {
	Published        bool               `json:"published"`
	CookeTimeMinutes uint32             `json:"cook_time_minutes"`
	Servings         float32            `json:"servings"`
	UserID           int64              `json:"user_id"`
	ID               int64              `json:"id"`
	CreatedAt        time.Time          `json:"created_at"`
	UpdatedAt        time.Time          `json:"updated_at"`
	ImageURL         string             `json:"image_url,omitempty"`
	Title            string             `json:"title"`
	Description      string             `json:"description,omitempty"`
	Ingredients      []RecipeIngredient `json:"ingredients"`
	Steps            []RecipeStep       `json:"steps"`
}

type RecipeWithIngredientsAndStepsAndOwner struct {
	Recipe RecipeWithIngredientsAndSteps `json:"recipe"`
	Owner  RecipeOwner                   `json:"owner"`
}

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
