// Package token contains utilities for http tokens.
package token

import "github.com/matt-dz/wecook/internal/env"

func AccessTokenName(env *env.Env) string {
	if env.Get("ENV") == "production" {
		return "__Host-Http-access"
	}
	return "access"
}
