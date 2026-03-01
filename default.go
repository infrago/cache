package cache

import (
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/infrago/infra"
	"github.com/coocood/freecache"
)

type defaultDriver struct{}

type defaultConnection struct {
	cache *freecache.Cache
	mu    sync.Mutex
}

func init() {
	module.RegisterDriver(infra.DEFAULT, &defaultDriver{})
}

func (d *defaultDriver) Connect(inst *Instance) (Connect, error) {
	// default 64MB
	size := 64 * 1024 * 1024
	if v, ok := inst.Config.Setting["size"].(int); ok && v > 0 {
		size = v
	}
	if v, ok := inst.Config.Setting["size"].(int64); ok && v > 0 {
		size = int(v)
	}
	return &defaultConnection{cache: freecache.NewCache(size)}, nil
}

func (c *defaultConnection) Open() error  { return nil }
func (c *defaultConnection) Close() error { return nil }

func (c *defaultConnection) Read(key string) ([]byte, error) {
	return c.cache.Get([]byte(key))
}

func (c *defaultConnection) Write(key string, val []byte, expire time.Duration) error {
	sec := int(expire.Seconds())
	if sec < 0 {
		sec = 0
	}
	return c.cache.Set([]byte(key), val, sec)
}

func (c *defaultConnection) Exists(key string) (bool, error) {
	_, err := c.cache.Get([]byte(key))
	if err != nil {
		return false, nil
	}
	return true, nil
}

func (c *defaultConnection) Delete(key string) error {
	c.cache.Del([]byte(key))
	return nil
}

func (c *defaultConnection) Sequence(key string, start, step int64, expire time.Duration) (int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	sec := int(expire.Seconds())
	if sec < 0 {
		sec = 0
	}

	var current int64
	if data, err := c.cache.Get([]byte(key)); err == nil {
		if v, err := strconv.ParseInt(string(data), 10, 64); err == nil {
			current = v
		}
	}

	if current == 0 {
		current = start
	} else {
		current += step
	}

	_ = c.cache.Set([]byte(key), []byte(strconv.FormatInt(current, 10)), sec)
	return current, nil
}

func (c *defaultConnection) Keys(prefix string) ([]string, error) {
	keys := make([]string, 0)
	it := c.cache.NewIterator()
	for {
		e := it.Next()
		if e == nil {
			break
		}
		k := string(e.Key)
		if prefix == "" || strings.HasPrefix(k, prefix) {
			keys = append(keys, k)
		}
	}
	return keys, nil
}

func (c *defaultConnection) Clear(prefix string) error {
	if prefix == "" {
		c.cache.Clear()
		return nil
	}
	it := c.cache.NewIterator()
	for {
		e := it.Next()
		if e == nil {
			break
		}
		k := string(e.Key)
		if strings.HasPrefix(k, prefix) {
			c.cache.Del(e.Key)
		}
	}
	return nil
}
