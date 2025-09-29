package store

import "errors"

// TODO - find an appropriate home for this
var (
	ErrNotFound = errors.New("does not exist")
	ErrExist    = errors.New("already exists")
)
