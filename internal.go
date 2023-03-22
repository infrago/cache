package cache

import (
	"time"

	. "github.com/infrago/base"
	"github.com/infrago/infra"
)

func (this *Module) Exists(key string) (bool, error) {
	locate := this.hashring.Locate(key)

	if inst, ok := this.instances[locate]; ok {
		key := inst.Config.Prefix + key //加前缀
		return inst.connect.Exists(key)
	}

	return false, errInvalidCacheConnection
}

func (this *Module) Read(key string) (Map, error) {
	locate := this.hashring.Locate(key)

	if inst, ok := this.instances[locate]; ok {
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

	return nil, errInvalidCacheConnection
}

// ReadData 读原始数据
func (this *Module) ReadData(key string) ([]byte, error) {
	locate := this.hashring.Locate(key)

	if inst, ok := this.instances[locate]; ok {
		realkey := inst.Config.Prefix + key
		return inst.connect.Read(realkey)
	}

	return nil, errInvalidCacheConnection
}

// Write 写缓存
func (this *Module) Write(key string, val Map, expiries ...time.Duration) error {
	locate := this.hashring.Locate(key)

	if inst, ok := this.instances[locate]; ok {
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

	return errInvalidCacheConnection
}

func (this *Module) WriteData(key string, data []byte, expiries ...time.Duration) error {
	locate := this.hashring.Locate(key)

	if inst, ok := this.instances[locate]; ok {
		expiry := inst.Config.Expiry
		if len(expiries) > 0 {
			expiry = expiries[0]
		}

		//KEY加上前缀
		realkey := inst.Config.Prefix + key
		return inst.connect.Write(realkey, data, expiry)
	}

	return errInvalidCacheConnection
}

// Delete 删除缓存
func (this *Module) Delete(key string) error {
	locate := this.hashring.Locate(key)

	if inst, ok := this.instances[locate]; ok {
		key := inst.Config.Prefix + key
		return inst.connect.Delete(key)
	}

	return errInvalidCacheConnection
}

// Serial 生成序列编号
func (this *Module) Serial(key string, start, step int64, expiries ...time.Duration) (int64, error) {
	locate := this.hashring.Locate(key)

	if inst, ok := this.instances[locate]; ok {
		expiry := time.Duration(0) //默认不过期
		if len(expiries) > 0 {
			expiry = expiries[0]
		}

		key := inst.Config.Prefix + key
		return inst.connect.Serial(key, start, step, expiry)
	}

	return -1, errInvalidCacheConnection
}

// Keys 获取所有前缀的KEYS
func (this *Module) Keys(prefix string) ([]string, error) {
	keys := make([]string, 0)

	for _, inst := range this.instances {
		realPrefix := inst.Config.Prefix + prefix
		temps, err := inst.connect.Keys(realPrefix)
		if err == nil {
			keys = append(keys, temps...)
		}
	}

	return keys, nil
}

// Clear 按前缀清理缓存
func (this *Module) Clear(prefix string) error {
	for _, inst := range this.instances {
		realPrefix := inst.Config.Prefix + prefix
		inst.connect.Clear(realPrefix)
	}

	return errInvalidCacheConnection
}
