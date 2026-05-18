package cache

import (
	"errors"
	"testing"
	"time"

	. "github.com/infrago/base"
	"github.com/infrago/infra"
)

func resetModuleForTest(t *testing.T) {
	t.Helper()
	module.Close()
	module.mutex.Lock()
	defer module.mutex.Unlock()
	module.configs = make(map[string]Config, 0)
	module.instances = make(map[string]*Instance, 0)
	module.weights = make(map[string]int, 0)
	module.hashring = nil
	module.opened = false
	module.stats.reset()
}

func TestDefaultCacheBasic(t *testing.T) {
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

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
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

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
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

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

func TestCacheConfigFromMap(t *testing.T) {
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

	module.Config(Map{
		"cache": Map{
			"driver":  "default",
			"prefix":  "app:",
			"codec":   infra.JSON,
			"expire":  "2s",
			"setting": Map{"size": 1024 * 1024},
			"hot": Map{
				"driver": "default",
				"prefix": "hot:",
				"ttl":    int64(5),
			},
		},
	})

	if got := module.configs[infra.DEFAULT].Prefix; got != "app:" {
		t.Fatalf("default prefix: %q", got)
	}
	if got := module.configs[infra.DEFAULT].Expire; got != 2*time.Second {
		t.Fatalf("default expire: %s", got)
	}
	if got := module.configs["hot"].Prefix; got != "hot:" {
		t.Fatalf("hot prefix: %q", got)
	}
	if got := module.configs["hot"].Expire; got != 5*time.Second {
		t.Fatalf("hot expire: %s", got)
	}
}

func TestCacheConfigFromPlainMap(t *testing.T) {
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

	module.Config(map[string]any{
		"cache": map[string]any{
			"driver": "default",
			"prefix": "plain:",
			"setting": map[string]any{
				"size": 1024 * 1024,
			},
		},
	})

	if got := module.configs[infra.DEFAULT].Prefix; got != "plain:" {
		t.Fatalf("plain map prefix: %q", got)
	}
	if got := module.configs[infra.DEFAULT].Setting["size"]; got != 1024*1024 {
		t.Fatalf("plain map setting: %v", got)
	}
}

func TestDefaultCacheMiss(t *testing.T) {
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

	module.Open()
	defer module.Close()

	out, err := module.Read("missing")
	if err != nil {
		t.Fatalf("read missing: %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil map on miss, got %v", out)
	}
	data, err := module.ReadData("missing")
	if err != nil {
		t.Fatalf("read data missing: %v", err)
	}
	if data != nil {
		t.Fatalf("expected nil data on miss, got %q", string(data))
	}
}

func TestCacheExistsUsesPrefix(t *testing.T) {
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

	module.RegisterConfig(infra.DEFAULT, Config{
		Driver:  infra.DEFAULT,
		Codec:   infra.JSON,
		Weight:  1,
		Prefix:  "app:",
		Setting: Map{"size": 1024 * 1024},
	})
	module.Open()
	defer module.Close()

	if err := module.Write("k", Map{"v": 1}); err != nil {
		t.Fatalf("write: %v", err)
	}
	ok, err := module.Exists("k")
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !ok {
		t.Fatal("expected prefixed key to exist")
	}
	ok, err = module.ExistsIn(infra.DEFAULT, "k")
	if err != nil {
		t.Fatalf("exists in: %v", err)
	}
	if !ok {
		t.Fatal("expected prefixed key to exist in default")
	}
}

func TestDefaultCacheSequenceStartZero(t *testing.T) {
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

	module.Open()
	defer module.Close()

	val, err := module.Sequence("seq0", 0, 1, time.Minute)
	if err != nil {
		t.Fatalf("seq: %v", err)
	}
	if val != 0 {
		t.Fatalf("expected 0, got %d", val)
	}
	val, err = module.Sequence("seq0", 0, 1, time.Minute)
	if err != nil {
		t.Fatalf("seq2: %v", err)
	}
	if val != 1 {
		t.Fatalf("expected 1, got %d", val)
	}
}

func TestDefaultCacheSequenceMany(t *testing.T) {
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

	module.Open()
	defer module.Close()

	vals, err := module.SequenceMany("seqMany", 10, 2, 3, time.Minute)
	if err != nil {
		t.Fatalf("sequence many: %v", err)
	}
	want := []int64{10, 12, 14}
	for i := range want {
		if vals[i] != want[i] {
			t.Fatalf("sequence many[%d]: expected %d, got %d", i, want[i], vals[i])
		}
	}
	next, err := module.Sequence("seqMany", 10, 2, time.Minute)
	if err != nil {
		t.Fatalf("sequence next: %v", err)
	}
	if next != 16 {
		t.Fatalf("expected next 16, got %d", next)
	}
}

func TestCacheNegativeWeightIsExcludedFromDistribution(t *testing.T) {
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

	module.RegisterConfig("active", Config{
		Driver:  infra.DEFAULT,
		Codec:   infra.JSON,
		Weight:  1,
		Setting: Map{"size": 1024 * 1024},
	})
	module.RegisterConfig("standby", Config{
		Driver:  infra.DEFAULT,
		Codec:   infra.JSON,
		Weight:  -1,
		Setting: Map{"size": 1024 * 1024},
	})
	module.Open()
	defer module.Close()

	if module.configs["standby"].Weight != -1 {
		t.Fatalf("expected negative weight to be preserved, got %d", module.configs["standby"].Weight)
	}
	for i := 0; i < 100; i++ {
		if got := module.hashring.Locate("key"); got != "active" {
			t.Fatalf("expected only active instance in distribution, got %q", got)
		}
	}
}

func TestCacheStats(t *testing.T) {
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

	module.Open()
	defer module.Close()

	_, _ = module.Read("missing")
	_ = module.Write("hit", Map{"v": 1})
	_, _ = module.Read("hit")
	_, _ = module.Exists("hit")
	_ = module.Delete("hit")
	_, _ = module.Keys()
	_ = module.Clear("hit")

	stats := module.Stats()
	if stats.Read != 2 || stats.Write != 1 || stats.Delete != 1 || stats.Exists != 1 || stats.Keys != 1 || stats.Clear != 1 {
		t.Fatalf("unexpected stats: %+v", stats)
	}
	if stats.Hit != 2 || stats.Miss != 1 || stats.Error != 0 {
		t.Fatalf("unexpected hit/miss/error stats: %+v", stats)
	}
	if stats.Instances[infra.DEFAULT].Read != 2 || stats.Instances[infra.DEFAULT].Hit != 2 {
		t.Fatalf("unexpected instance stats: %+v", stats.Instances[infra.DEFAULT])
	}
	instStats, err := module.StatsFrom(infra.DEFAULT)
	if err != nil {
		t.Fatalf("stats from: %v", err)
	}
	if instStats.Write != 1 {
		t.Fatalf("unexpected stats from: %+v", instStats)
	}
}

func TestCacheKeyBuilder(t *testing.T) {
	if got := Key("user", 12, "profile"); got != "user:12:profile" {
		t.Fatalf("unexpected key: %q", got)
	}
	if got := KeyWith("/", "user", 12, "profile"); got != "user/12/profile" {
		t.Fatalf("unexpected key with sep: %q", got)
	}
}

func TestCacheClearAllSafety(t *testing.T) {
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

	module.Open()
	defer module.Close()

	if err := module.Write("danger", Map{"v": 1}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := module.Clear(); !errors.Is(err, ErrUnsafeClear) {
		t.Fatalf("expected unsafe clear error, got %v", err)
	}
	ok, err := module.Exists("danger")
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !ok {
		t.Fatal("unsafe clear removed key")
	}
	if err := module.ClearAll(); err != nil {
		t.Fatalf("clear all: %v", err)
	}
	ok, err = module.Exists("danger")
	if err != nil {
		t.Fatalf("exists2: %v", err)
	}
	if ok {
		t.Fatal("clear all did not remove key")
	}
}

func TestCacheClearAllAllowedByConfig(t *testing.T) {
	resetModuleForTest(t)
	t.Cleanup(func() { resetModuleForTest(t) })

	module.RegisterConfig(infra.DEFAULT, Config{
		Driver:        infra.DEFAULT,
		Codec:         infra.JSON,
		Weight:        1,
		AllowClearAll: true,
		Setting:       Map{"size": 1024 * 1024},
	})
	module.Open()
	defer module.Close()

	if err := module.Write("danger", Map{"v": 1}); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := module.Clear(); err != nil {
		t.Fatalf("clear: %v", err)
	}
	ok, err := module.Exists("danger")
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if ok {
		t.Fatal("allowed clear all did not remove key")
	}
}
