package cache

import (
	"time"

	. "github.com/infrago/base"
)

func Read(key string) (Map, error) {
	return module.Read(key)
}
func ReadData(key string) ([]byte, error) {
	return module.ReadData(key)
}
func Write(key string, value Map, expiries ...time.Duration) error {
	return module.Write(key, value, expiries...)
}
func WriteData(key string, data []byte, expiries ...time.Duration) error {
	return module.WriteData(key, data, expiries...)
}
func Delete(key string) error {
	return module.Delete(key)
}
func Exists(key string) (bool, error) {
	return module.Exists(key)
}
func Sequence(key string, start, step int64, expiries ...time.Duration) (int64, error) {
	return module.Sequence(key, start, step, expiries...)
}
func Keys(prefixs ...string) ([]string, error) {
	prefix := ""
	if len(prefixs) > 0 {
		prefix = prefixs[0]
	}
	return module.Keys(prefix)
}
func Clear(prefixs ...string) error {
	prefix := ""
	if len(prefixs) > 0 {
		prefix = prefixs[0]
	}

	return module.Clear(prefix)
}
