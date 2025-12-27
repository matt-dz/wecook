// Package http provides a wrapper around the retryablehttp.Client
// for making HTTP requests with retry capabilities.
package http

import (
	"fmt"
	"io"
	"net/http"

	"github.com/hashicorp/go-retryablehttp"
)

type HTTPDoer interface {
	Do(*retryablehttp.Request) (*http.Response, error)
}

type HTTP struct {
	*retryablehttp.Client
}

var _ HTTPDoer = (*retryablehttp.Client)(nil)

func DefaultConfig() *retryablehttp.Client {
	return retryablehttp.NewClient()
}

func New(client *retryablehttp.Client) *HTTP {
	return &HTTP{
		Client: client,
	}
}

func ExpectStatus2xx(resp *http.Response) error {
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}
