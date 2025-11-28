// Package garage contains utility functions for garage.
package garage

import (
	"errors"

	garage "git.deuxfleurs.fr/garage-sdk/garage-admin-sdk-golang"
	"github.com/matt-dz/wecook/internal/env"
)

func NewClient(env *env.Env) (*garage.APIClient, error) {
	config := garage.NewConfiguration()

	// Set host
	host := env.Get("GARAGE_ADMIN_URL")
	if host == "" {
		return nil, errors.New("environment variable GARAGE_ADMIN_URL must be set")
	}
	config.Host = host
	config.HTTPClient = env.HTTP.HTTPClient
	client := garage.NewAPIClient(config)

	// Check admin token
	adminToken := env.Get("GARAGE_ADMIN_TOKEN")
	if adminToken == "" {
		return nil, errors.New("environment variable GARAGE_ADMIN_TOKEN must be set")
	}

	return client, nil
}
