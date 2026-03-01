package cache

import (
	"fmt"
	"sync"
	"time"

	"github.com/infrago/infra"
	. "github.com/infrago/base"
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
	}

	Configs map[string]Config
	Config  struct {
		Driver  string
		Weight  int
		Prefix  string
		Codec   string
		Expire  time.Duration
		Setting Map
	}
	Instance struct {
		connect Connect
		Name    string
		Config  Config
		Setting Map
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

func (m *Module) Config(Map) {}
func (m *Module) Setup()     {}

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
		m.weights[name] = cfg.Weight
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
