package cache

import "errors"

const NAME = "CACHE"

var ErrInvalidConnection = errors.New("Invalid cache connection.")
var ErrUnsafeClear = errors.New("Unsafe cache clear blocked.")
