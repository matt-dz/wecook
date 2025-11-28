// Package http provides a wrapper around the retryablehttp.Client
// for making HTTP requests with retry capabilities.
package http

import (
	"github.com/hashicorp/go-retryablehttp"
)

type HTTP struct {
	*retryablehttp.Client
}

func New() *HTTP {
	client := retryablehttp.NewClient()
	return &HTTP{
		Client: client,
	}
}
