package cache

import (
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/coocood/freecache"
	"github.com/infrago/infra"
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
	data, err := c.cache.Get([]byte(key))
	if errors.Is(err, freecache.ErrNotFound) {
		return nil, nil
	}
	return data, err
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
	if errors.Is(err, freecache.ErrNotFound) {
		return false, nil
	}
	return err == nil, err
}

func (c *defaultConnection) Delete(key string) error {
	c.cache.Del([]byte(key))
	return nil
}

func (c *defaultConnection) Sequence(key string, start, step int64, expire time.Duration) (int64, error) {
	vals, err := c.SequenceMany(key, start, step, 1, expire)
	if err != nil {
		return -1, err
	}
	return vals[0], nil
}

func (c *defaultConnection) SequenceMany(key string, start, step, count int64, expire time.Duration) ([]int64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if count <= 0 {
		return []int64{}, nil
	}
	sec := int(expire.Seconds())
	if sec < 0 {
		sec = 0
	}

	current := start
	found := false
	if data, err := c.cache.Get([]byte(key)); err == nil {
		if v, err := strconv.ParseInt(string(data), 10, 64); err == nil {
			current = v
			found = true
		}
	}

	if found {
		current += step
	}

	vals := make([]int64, 0, count)
	for i := int64(0); i < count; i++ {
		if i > 0 {
			current += step
		}
		vals = append(vals, current)
	}

	_ = c.cache.Set([]byte(key), []byte(strconv.FormatInt(current, 10)), sec)
	return vals, nil
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
