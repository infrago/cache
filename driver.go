package cache

import (
	"time"
)

type (
	// Driver 数据驱动
	Driver interface {
		Connect(*Instance) (Connect, error)
	}

	// Connect 会话连接
	Connect interface {
		Open() error
		Close() error

		Read(string) ([]byte, error)
		Write(key string, val []byte, expiry time.Duration) error
		Exists(key string) (bool, error)
		Delete(key string) error
		Serial(key string, start, step int64, expiry time.Duration) (int64, error)
		Keys(prefix string) ([]string, error)
		Clear(prefix string) error
	}
)
