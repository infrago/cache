package cache

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	. "github.com/infrago/base"
	"github.com/infrago/infra"
	"github.com/infrago/util"
)

func init() {
	infra.Mount(module)
}

var (
	module = &Module{
		configs:   make(map[string]Config, 0),
		drivers:   make(map[string]Driver, 0),
		instances: make(map[string]*Instance, 0),
		weights:   make(map[string]int, 0),
	}
)

type (
	Module struct {
		mutex sync.Mutex

		opened bool

		configs   map[string]Config
		drivers   map[string]Driver
		instances map[string]*Instance

		weights  map[string]int
		hashring *util.HashRing

		stats cacheStats
	}

	Configs map[string]Config
	Config  struct {
		Driver        string
		Weight        int
		Prefix        string
		Codec         string
		Expire        time.Duration
		AllowClearAll bool
		Setting       Map
	}
	Instance struct {
		connect Connect
		Name    string
		Config  Config
		Setting Map
		stats   cacheStats
	}
	Statistics struct {
		Read   uint64
		Write  uint64
		Delete uint64
		Exists uint64
		Keys   uint64
		Clear  uint64

		Sequence     uint64
		SequenceMany uint64

		Hit   uint64
		Miss  uint64
		Error uint64

		Instances map[string]Statistics
	}
	cacheStats struct {
		read         atomic.Uint64
		write        atomic.Uint64
		delete       atomic.Uint64
		exists       atomic.Uint64
		keys         atomic.Uint64
		clear        atomic.Uint64
		sequence     atomic.Uint64
		sequenceMany atomic.Uint64
		hit          atomic.Uint64
		miss         atomic.Uint64
		error        atomic.Uint64
	}
)

func (m *Module) Register(name string, value Any) {
	switch v := value.(type) {
	case Driver:
		m.RegisterDriver(name, v)
	case Config:
		m.RegisterConfig(name, v)
	case Configs:
		m.RegisterConfigs(v)
	}
}

func (m *Module) RegisterDriver(name string, driver Driver) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if name == "" {
		name = infra.DEFAULT
	}
	if driver == nil {
		panic("Invalid cache driver: " + name)
	}
	if infra.Override() {
		m.drivers[name] = driver
	} else {
		if _, ok := m.drivers[name]; !ok {
			m.drivers[name] = driver
		}
	}
}

func (m *Module) RegisterConfig(name string, config Config) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if name == "" {
		name = infra.DEFAULT
	}
	if infra.Override() {
		m.configs[name] = config
	} else {
		if _, ok := m.configs[name]; !ok {
			m.configs[name] = config
		}
	}
}

func (m *Module) RegisterConfigs(configs Configs) {
	for key, val := range configs {
		m.RegisterConfig(key, val)
	}
}

func (m *Module) Config(global Map) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.opened {
		return
	}

	cfgAny, ok := global["cache"]
	if !ok {
		return
	}
	cfgMap, ok := castMap(cfgAny)
	if !ok || cfgMap == nil {
		return
	}

	root := Map{}
	for key, val := range cfgMap {
		if conf, ok := castMap(val); ok && key != "setting" {
			m.configure(key, conf)
		} else {
			root[key] = val
		}
	}
	if len(root) > 0 {
		m.configure(infra.DEFAULT, root)
	}
}

func (m *Module) configure(name string, conf Map) {
	cfg := Config{Driver: infra.DEFAULT, Codec: infra.JSON, Weight: 1}
	if existed, ok := m.configs[name]; ok {
		cfg = existed
	}

	if v, ok := conf["driver"].(string); ok && v != "" {
		cfg.Driver = v
	}
	if v, ok := conf["prefix"].(string); ok {
		cfg.Prefix = v
	}
	if v, ok := conf["codec"].(string); ok && v != "" {
		cfg.Codec = v
	}
	if v, ok := parseInt(conf["weight"]); ok {
		cfg.Weight = v
	}
	if v, ok := parseDuration(conf["expire"]); ok {
		cfg.Expire = v
	}
	if v, ok := parseDuration(conf["ttl"]); ok {
		cfg.Expire = v
	}
	if v, ok := parseBool(conf["allow_clear_all"]); ok {
		cfg.AllowClearAll = v
	}
	if v, ok := parseBool(conf["clear_all"]); ok {
		cfg.AllowClearAll = v
	}
	if v, ok := castMap(conf["setting"]); ok {
		cfg.Setting = v
	}

	m.configs[name] = cfg
}

func (m *Module) Setup() {}

func (m *Module) Open() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if m.opened {
		return
	}

	if len(m.configs) == 0 {
		m.configs[infra.DEFAULT] = Config{Driver: infra.DEFAULT, Weight: 1}
	}

	for name, cfg := range m.configs {
		if name == "" {
			name = infra.DEFAULT
		}
		if cfg.Driver == "" {
			cfg.Driver = infra.DEFAULT
		}
		if cfg.Codec == "" {
			cfg.Codec = infra.JSON
		}
		if cfg.Weight == 0 {
			cfg.Weight = 1
		}
		m.configs[name] = cfg
	}

	for name, cfg := range m.configs {
		driver, ok := m.drivers[cfg.Driver]
		if !ok || driver == nil {
			panic("Missing cache driver: " + cfg.Driver)
		}

		inst := &Instance{Name: name, Config: cfg, Setting: cfg.Setting}
		conn, err := driver.Connect(inst)
		if err != nil {
			panic("Failed to connect cache: " + err.Error())
		}
		if err := conn.Open(); err != nil {
			panic("Failed to open cache: " + err.Error())
		}
		inst.connect = conn
		m.instances[name] = inst
		if cfg.Weight > 0 {
			m.weights[name] = cfg.Weight
		}
	}

	m.hashring = util.NewHashRing(m.weights)
	m.opened = true
}

func (m *Module) Start() {
	fmt.Printf("infrago cache module is running with %d connections.\n", len(m.instances))
}

func (m *Module) Stop() {}

func (m *Module) Close() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if !m.opened {
		return
	}

	for _, inst := range m.instances {
		_ = inst.connect.Close()
	}

	m.instances = make(map[string]*Instance, 0)
	m.weights = make(map[string]int, 0)
	m.hashring = nil
	m.opened = false
}

func (m *Module) Stats() Statistics {
	out := m.stats.snapshot()
	out.Instances = map[string]Statistics{}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	for name, inst := range m.instances {
		out.Instances[name] = inst.stats.snapshot()
	}
	return out
}

func (m *Module) StatsFrom(conn string) (Statistics, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	inst, ok := m.instances[conn]
	if !ok {
		return Statistics{}, ErrInvalidConnection
	}
	return inst.stats.snapshot(), nil
}

func (m *Module) ResetStats() {
	m.stats.reset()

	m.mutex.Lock()
	defer m.mutex.Unlock()
	for _, inst := range m.instances {
		inst.stats.reset()
	}
}

func (s *cacheStats) snapshot() Statistics {
	return Statistics{
		Read:         s.read.Load(),
		Write:        s.write.Load(),
		Delete:       s.delete.Load(),
		Exists:       s.exists.Load(),
		Keys:         s.keys.Load(),
		Clear:        s.clear.Load(),
		Sequence:     s.sequence.Load(),
		SequenceMany: s.sequenceMany.Load(),
		Hit:          s.hit.Load(),
		Miss:         s.miss.Load(),
		Error:        s.error.Load(),
	}
}

func (s *cacheStats) reset() {
	s.read.Store(0)
	s.write.Store(0)
	s.delete.Store(0)
	s.exists.Store(0)
	s.keys.Store(0)
	s.clear.Store(0)
	s.sequence.Store(0)
	s.sequenceMany.Store(0)
	s.hit.Store(0)
	s.miss.Store(0)
	s.error.Store(0)
}

func parseInt(v Any) (int, bool) {
	switch vv := v.(type) {
	case int:
		return vv, true
	case int8:
		return int(vv), true
	case int16:
		return int(vv), true
	case int32:
		return int(vv), true
	case int64:
		return int(vv), true
	case uint:
		return int(vv), true
	case uint8:
		return int(vv), true
	case uint16:
		return int(vv), true
	case uint32:
		return int(vv), true
	case uint64:
		return int(vv), true
	case float32:
		return int(vv), true
	case float64:
		return int(vv), true
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(vv))
		return n, err == nil
	default:
		return 0, false
	}
}

func parseBool(v Any) (bool, bool) {
	switch vv := v.(type) {
	case bool:
		return vv, true
	case int:
		return vv != 0, true
	case int8:
		return vv != 0, true
	case int16:
		return vv != 0, true
	case int32:
		return vv != 0, true
	case int64:
		return vv != 0, true
	case uint:
		return vv != 0, true
	case uint8:
		return vv != 0, true
	case uint16:
		return vv != 0, true
	case uint32:
		return vv != 0, true
	case uint64:
		return vv != 0, true
	case string:
		switch strings.ToLower(strings.TrimSpace(vv)) {
		case "true", "1", "yes", "on":
			return true, true
		case "false", "0", "no", "off":
			return false, true
		default:
			return false, false
		}
	default:
		return false, false
	}
}

func castMap(v Any) (Map, bool) {
	switch vv := v.(type) {
	case Map:
		return vv, true
	}
	rv := reflect.ValueOf(v)
	if !rv.IsValid() || rv.Kind() != reflect.Map || rv.Type().Key().Kind() != reflect.String {
		return nil, false
	}
	out := Map{}
	iter := rv.MapRange()
	for iter.Next() {
		out[iter.Key().String()] = iter.Value().Interface()
	}
	return out, true
}

func parseDuration(v Any) (time.Duration, bool) {
	switch vv := v.(type) {
	case time.Duration:
		return vv, true
	case int:
		return time.Duration(vv) * time.Second, true
	case int8:
		return time.Duration(vv) * time.Second, true
	case int16:
		return time.Duration(vv) * time.Second, true
	case int32:
		return time.Duration(vv) * time.Second, true
	case int64:
		return time.Duration(vv) * time.Second, true
	case uint:
		return time.Duration(vv) * time.Second, true
	case uint8:
		return time.Duration(vv) * time.Second, true
	case uint16:
		return time.Duration(vv) * time.Second, true
	case uint32:
		return time.Duration(vv) * time.Second, true
	case uint64:
		return time.Duration(vv) * time.Second, true
	case float32:
		return time.Duration(vv * float32(time.Second)), true
	case float64:
		return time.Duration(vv * float64(time.Second)), true
	case string:
		d, err := time.ParseDuration(strings.TrimSpace(vv))
		return d, err == nil
	default:
		return 0, false
	}
}
