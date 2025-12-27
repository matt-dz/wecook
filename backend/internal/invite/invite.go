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
func EncodeInvite(code string, id int64) string {
	return fmt.Sprintf("%d%c%s", id, delimiter, code)
}

// DecodeInvite decodes an invite.
func DecodeInvite(invite string) (code string, id int64, err error) {
	idStr, code, found := strings.Cut(code, string(delimiter))
	if !found {
		return "", 0, ErrInvalidInvite
	}

	id, err = strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return "", 0, errors.Join(ErrInvalidInvite, err)
	}

	return code, id, nil
}
