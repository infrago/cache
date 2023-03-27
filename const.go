package cache

import "errors"

const (
	NAME = "CACHE"
)

var (
	ErrInvalidConnection = errors.New("Invalid cache connection.")
)
