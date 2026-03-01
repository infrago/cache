package cache

import (
	"encoding/json"
	"strings"
	"time"

	. "github.com/infrago/base"
	"github.com/infrago/infra"
)

func (m *Module) getInst(conn, key string) (*Instance, error) {
	if conn == "" {
		if m.hashring == nil {
			return nil, ErrInvalidConnection
		}
		conn = m.hashring.Locate(key)
	}
	if inst, ok := m.instances[conn]; ok {
		return inst, nil
	}
	return nil, ErrInvalidConnection
}

func (m *Module) Exists(key string) (bool, error) {
	inst, err := m.getInst("", key)
	if err != nil {
		return false, err
	}
	return inst.connect.Exists(key)
}

func (m *Module) ExistsIn(conn, key string) (bool, error) {
	inst, err := m.getInst(conn, key)
	if err != nil {
		return false, err
	}
	return inst.connect.Exists(key)
}

func (m *Module) ReadFrom(conn, key string) (Map, error) {
	inst, err := m.getInst(conn, key)
	if err != nil {
		return nil, err
	}
	realKey := inst.Config.Prefix + key
	data, err := inst.connect.Read(realKey)
	if err != nil {
		return nil, err
	}
	val := Map{}
	if err := cacheUnmarshal(inst.Config.Codec, data, &val); err != nil {
		return nil, err
	}
	return val, nil
}

func (m *Module) Read(key string) (Map, error) {
	return m.ReadFrom("", key)
}

func (m *Module) ReadDataFrom(conn, key string) ([]byte, error) {
	inst, err := m.getInst(conn, key)
	if err != nil {
		return nil, err
	}
	realKey := inst.Config.Prefix + key
	return inst.connect.Read(realKey)
}

func (m *Module) ReadData(key string) ([]byte, error) {
	return m.ReadDataFrom("", key)
}

func (m *Module) WriteTo(conn string, key string, val Map, expires ...time.Duration) error {
	inst, err := m.getInst(conn, key)
	if err != nil {
		return err
	}

	expire := inst.Config.Expire
	if len(expires) > 0 {
		expire = expires[0]
	}

	data, err := cacheMarshal(inst.Config.Codec, &val)
	if err != nil {
		return err
	}

	realKey := inst.Config.Prefix + key
	return inst.connect.Write(realKey, data, expire)
}

func (m *Module) Write(key string, val Map, expires ...time.Duration) error {
	return m.WriteTo("", key, val, expires...)
}

func (m *Module) WriteDataTo(conn, key string, data []byte, expires ...time.Duration) error {
	inst, err := m.getInst(conn, key)
	if err != nil {
		return err
	}

	expire := inst.Config.Expire
	if len(expires) > 0 {
		expire = expires[0]
	}

	realKey := inst.Config.Prefix + key
	return inst.connect.Write(realKey, data, expire)
}

func (m *Module) WriteData(key string, data []byte, expires ...time.Duration) error {
	return m.WriteDataTo("", key, data, expires...)
}

func (m *Module) DeleteFrom(conn, key string) error {
	inst, err := m.getInst(conn, key)
	if err != nil {
		return err
	}
	realKey := inst.Config.Prefix + key
	return inst.connect.Delete(realKey)
}

func (m *Module) Delete(key string) error {
	return m.DeleteFrom("", key)
}

func (m *Module) SequenceOn(conn, key string, start, step int64, expires ...time.Duration) (int64, error) {
	inst, err := m.getInst(conn, key)
	if err != nil {
		return -1, err
	}
	expire := time.Duration(0)
	if len(expires) > 0 {
		expire = expires[0]
	}
	realKey := inst.Config.Prefix + key
	return inst.connect.Sequence(realKey, start, step, expire)
}

func (m *Module) Sequence(key string, start, step int64, expires ...time.Duration) (int64, error) {
	return m.SequenceOn("", key, start, step, expires...)
}

func (m *Module) KeysFrom(conn string, prefixs ...string) ([]string, error) {
	prefix := ""
	if len(prefixs) > 0 {
		prefix = prefixs[0]
	}

	keys := make([]string, 0)

	if conn == "" {
		for _, inst := range m.instances {
			realPrefix := inst.Config.Prefix + prefix
			temps, err := inst.connect.Keys(realPrefix)
			if err == nil {
				for _, temp := range temps {
					keys = append(keys, strings.TrimPrefix(temp, realPrefix))
				}
			}
		}
		return keys, nil
	}

	if inst, ok := m.instances[conn]; ok {
		realPrefix := inst.Config.Prefix + prefix
		temps, err := inst.connect.Keys(realPrefix)
		if err == nil {
			for _, temp := range temps {
				keys = append(keys, strings.TrimPrefix(temp, realPrefix))
			}
		}
		return keys, nil
	}

	return keys, ErrInvalidConnection
}

func (m *Module) Keys(prefixs ...string) ([]string, error) {
	return m.KeysFrom("", prefixs...)
}

func (m *Module) ClearFrom(conn string, prefixs ...string) error {
	prefix := ""
	if len(prefixs) > 0 {
		prefix = prefixs[0]
	}

	if conn == "" {
		for _, inst := range m.instances {
			realPrefix := inst.Config.Prefix + prefix
			_ = inst.connect.Clear(realPrefix)
		}
		return nil
	}

	if inst, ok := m.instances[conn]; ok {
		realPrefix := inst.Config.Prefix + prefix
		return inst.connect.Clear(realPrefix)
	}

	return ErrInvalidConnection
}

func (m *Module) Clear(prefixs ...string) error {
	return m.ClearFrom("", prefixs...)
}

func cacheMarshal(codec string, value Any) ([]byte, error) {
	name := strings.TrimSpace(codec)
	if name == "" {
		name = infra.JSON
	}
	data, err := infra.Marshal(name, value)
	if err == nil {
		return data, nil
	}
	if strings.EqualFold(name, infra.JSON) && err.Error() == "Invalid codec." {
		return json.Marshal(value)
	}
	return nil, err
}

func cacheUnmarshal(codec string, data []byte, value Any) error {
	name := strings.TrimSpace(codec)
	if name == "" {
		name = infra.JSON
	}
	err := infra.Unmarshal(name, data, value)
	if err == nil {
		return nil
	}
	if strings.EqualFold(name, infra.JSON) && err.Error() == "Invalid codec." {
		return json.Unmarshal(data, value)
	}
	return err
}
