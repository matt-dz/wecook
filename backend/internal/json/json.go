// Package json contains utilities for handling JSON.
package json

import (
	"encoding/json"
	"fmt"
	"io"
)

// DecodeJSON decodes a JSON object.
func DecodeJSON(dst any, decoder *json.Decoder) error {
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("decoding json: %w", err)
	}

	// Ensure no extra tokens after decoding
	if _, err := decoder.Token(); err != io.EOF {
		return fmt.Errorf("unexpected token after JSON object: %w", err)
	}
	return nil
}
