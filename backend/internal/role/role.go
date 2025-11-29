// Package role contains utilities for user roles.
package role

import (
	"math"

	"github.com/matt-dz/wecook/internal/database"
)

type Role int

const (
	RoleAdmin   Role = 200
	RoleUser    Role = 100
	RoleUnknown Role = math.MinInt
)

func (r Role) String() string {
	switch r {
	case RoleAdmin:
		return "admin"
	case RoleUser:
		return "user"
	default:
		return "unknown"
	}
}

func DBToRole(role database.Role) Role {
	switch role {
	case database.RoleAdmin:
		return RoleAdmin
	case database.RoleUser:
		return RoleUser
	default:
		return RoleUnknown
	}
}

func ToRole(role string) Role {
	switch role {
	case "admin":
		return RoleAdmin
	case "user":
		return RoleUser
	default:
		return RoleUnknown
	}
}
