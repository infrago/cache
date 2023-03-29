package cache

import (
	"time"

	. "github.com/infrago/base"
	"github.com/infrago/infra"
)

func (this *Module) getInst(conn, key string) (*Instance, error) {
	if conn == "" {
		conn = this.hashring.Locate(key)
	}
	if inst, ok := this.instances[conn]; ok {
		return inst, nil
	}
	return nil, ErrInvalidConnection
}

// Exists
// 判断缓存是否存在
func (this *Module) Exists(key string) (bool, error) {
	inst, err := this.getInst("", key)
	if err != nil {
		return false, err
	}
	return inst.connect.Exists(key)
}

// ExistsIn
// 判断缓存是否存在指定库
func (this *Module) ExistsIn(conn, key string) (bool, error) {
	inst, err := this.getInst(conn, key)
	if err != nil {
		return false, err
	}
	return inst.connect.Exists(key)
}

func (this *Module) ReadFrom(conn, key string) (Map, error) {
	inst, err := this.getInst(conn, key)
	if err != nil {
		return nil, err
	}
	//加前缀
	realkey := inst.Config.Prefix + key
	data, err := inst.connect.Read(realkey)
	if err != nil {
		return nil, err
	}

	val := Map{}
	err = infra.Unmarshal(inst.Config.Codec, data, &val)
	if err != nil {
		return nil, err
	}

	return val, nil
}

// Read 读取缓存
func (this *Module) Read(key string) (Map, error) {
	return this.ReadFrom("", key)
}

// ReadDataFrom 从指定库读原始数据
func (this *Module) ReadDataFrom(conn, key string) ([]byte, error) {
	inst, err := this.getInst(conn, key)
	if err != nil {
		return nil, err
	}
	//加前缀
	realkey := inst.Config.Prefix + key
	data, err := inst.connect.Read(realkey)
	if err != nil {
		return nil, err
	}

	return data, nil
}

// ReadData 读取原始数据
func (this *Module) ReadData(key string) ([]byte, error) {
	return this.ReadDataFrom("", key)
}

// Write 写缓存
func (this *Module) WriteTo(conn string, key string, val Map, expiries ...time.Duration) error {
	inst, err := this.getInst(conn, key)
	if err != nil {
		return err
	}

	//默认超时时间
	expiry := inst.Config.Expiry
	if len(expiries) > 0 {
		expiry = expiries[0]
	}

	// 编码数据
	data, err := infra.Marshal(inst.Config.Codec, &val)
	if err != nil {
		return err
	}

	//KEY加上前缀
	realkey := inst.Config.Prefix + key
	return inst.connect.Write(realkey, data, expiry)
}

// Write 写缓存
func (this *Module) Write(key string, val Map, expiries ...time.Duration) error {
	return this.WriteTo("", key, val, expiries...)
}

func (this *Module) WriteDataTo(conn, key string, data []byte, expiries ...time.Duration) error {
	inst, err := this.getInst(conn, key)
	if err != nil {
		return err
	}

	expiry := inst.Config.Expiry
	if len(expiries) > 0 {
		expiry = expiries[0]
	}

	//KEY加上前缀
	realkey := inst.Config.Prefix + key
	return inst.connect.Write(realkey, data, expiry)
}

// Write 写缓原始数据
func (this *Module) WriteData(key string, data []byte, expiries ...time.Duration) error {
	return this.WriteDataTo("", key, data, expiries...)
}

// Delete 从指定库删除缓存
func (this *Module) DeleteFrom(conn, key string) error {
	inst, err := this.getInst(conn, key)
	if err != nil {
		return err
	}

	realKey := inst.Config.Prefix + key
	return inst.connect.Delete(realKey)
}

// Delete 删除缓存
func (this *Module) Delete(key string) error {
	return this.DeleteFrom("", key)
}

// SequenceOn 指定库生成编号
func (this *Module) SequenceOn(conn, key string, start, step int64, expiries ...time.Duration) (int64, error) {
	inst, err := this.getInst(conn, key)
	if err != nil {
		return -1, err
	}

	expiry := time.Duration(0) //默认不过期
	if len(expiries) > 0 {
		expiry = expiries[0]
	}

	realKey := inst.Config.Prefix + key
	return inst.connect.Sequence(realKey, start, step, expiry)
}

// Sequence 生成编号
func (this *Module) Sequence(key string, start, step int64, expiries ...time.Duration) (int64, error) {
	return this.SequenceOn("", key, start, step, expiries...)
}

// Keys 获取所有前缀的KEYS
func (this *Module) KeysFrom(conn string, prefixs ...string) ([]string, error) {
	prefix := ""
	if len(prefixs) > 0 {
		prefix = prefixs[0]
	}

	keys := make([]string, 0)

	//全部库
	if conn == "" {
		for _, inst := range this.instances {
			realPrefix := inst.Config.Prefix + prefix
			temps, err := inst.connect.Keys(realPrefix)
			if err == nil {
				keys = append(keys, temps...)
			}
		}
		return keys, nil
	}

	//指定库
	for _, inst := range this.instances {
		realPrefix := inst.Config.Prefix + prefix
		temps, err := inst.connect.Keys(realPrefix)
		if err == nil {
			keys = append(keys, temps...)
		}
	}

	return keys, ErrInvalidConnection
}
func (this *Module) Keys(prefixs ...string) ([]string, error) {
	return this.KeysFrom("", prefixs...)
}

// Clear 按前缀清理缓存
func (this *Module) ClearFrom(conn string, prefixs ...string) error {
	prefix := ""
	if len(prefixs) > 0 {
		prefix = prefixs[0]
	}

	if conn == "" {
		for _, inst := range this.instances {
			realPrefix := inst.Config.Prefix + prefix
			inst.connect.Clear(realPrefix)
		}
		return nil
	}

	//指定库
	for _, inst := range this.instances {
		realPrefix := inst.Config.Prefix + prefix
		inst.connect.Clear(realPrefix)
	}

	return ErrInvalidConnection
}

// Clear 清里缓存
func (this *Module) Clear(prefixs ...string) error {
	return this.ClearFrom("", prefixs...)
}
