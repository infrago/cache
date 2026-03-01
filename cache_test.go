package cache

import (
	"testing"
	"time"

	. "github.com/infrago/base"
)

func TestDefaultCacheBasic(t *testing.T) {
	module.Open()
	defer module.Close()

	key := "k1"
	val := Map{"a": 1, "b": "x"}
	if err := module.Write(key, val); err != nil {
		t.Fatalf("write: %v", err)
	}
	out, err := module.Read(key)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if out["b"] != "x" {
		t.Fatalf("unexpected value: %v", out)
	}

	exists, err := module.Exists(key)
	if err != nil || !exists {
		t.Fatalf("exists: %v %v", exists, err)
	}

	if err := module.Delete(key); err != nil {
		t.Fatalf("delete: %v", err)
	}
}

func TestDefaultCacheKeysClear(t *testing.T) {
	module.Open()
	defer module.Close()

	_ = module.Write("p:1", Map{"v": 1})
	_ = module.Write("p:2", Map{"v": 2})
	_ = module.Write("q:1", Map{"v": 3})

	keys, err := module.Keys("p:")
	if err != nil {
		t.Fatalf("keys: %v", err)
	}
	if len(keys) < 2 {
		t.Fatalf("keys size: %v", keys)
	}

	if err := module.Clear("p:"); err != nil {
		t.Fatalf("clear: %v", err)
	}
}

func TestDefaultCacheSequence(t *testing.T) {
	module.Open()
	defer module.Close()

	key := "seq"
	val, err := module.Sequence(key, 100, 1, time.Minute)
	if err != nil {
		t.Fatalf("seq: %v", err)
	}
	if val != 100 {
		t.Fatalf("expected 100, got %d", val)
	}
	val, err = module.Sequence(key, 100, 1, time.Minute)
	if err != nil {
		t.Fatalf("seq2: %v", err)
	}
	if val != 101 {
		t.Fatalf("expected 101, got %d", val)
	}
}
