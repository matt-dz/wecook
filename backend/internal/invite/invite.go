// Package invite contains utilities for creating and managing invites.
package invite

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/matt-dz/wecook/internal/api/token"
)

const (
	inviteCodeBytes = 16
	delimiter       = '$'
)

var ErrInvalidInvite = errors.New("malformed invite")

// CreateInvite creates a cryptographically secure random invite.
func CreateInvite() (code string, err error) {
	return token.CreateToken(inviteCodeBytes)
}

// EncodeInvite encodes the invite to be sent to the user.
func EncodeInvite(id int64, code string) string {
	return fmt.Sprintf("%d%c%s", id, delimiter, code)
}

// DecodeInvite decodes an invite.
func DecodeInvite(invite string) (id int64, code string, err error) {
	idStr, code, found := strings.Cut(invite, string(delimiter))
	if !found {
		return id, code, ErrInvalidInvite
	}

	id, err = strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return id, code, errors.Join(ErrInvalidInvite, err)
	}

	return id, code, nil
}
