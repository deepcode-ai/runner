package store

import (
	"errors"

	"github.com/deepcode-ai/runner/auth/model"
)

var ErrEmpty = errors.New("store: empty")

type Store interface {
	SetAccessCode(code string, user *model.User) error
	VerifyAccessCode(code string) (*model.User, error)
}
