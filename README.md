# cache

`cache` 是 infrago 的模块包。

## 安装

```bash
go get github.com/infrago/cache@latest
```

## 最小接入

```go
package main

import (
    _ "github.com/infrago/cache"
    "github.com/infrago/infra"
)

func main() {
    infra.Run()
}
```

## 配置示例

```toml
[cache]
driver = "default"
```

## 公开 API（摘自源码）

- `func (m *Module) Exists(key string) (bool, error)`
- `func (m *Module) ExistsIn(conn, key string) (bool, error)`
- `func (m *Module) ReadFrom(conn, key string) (Map, error)`
- `func (m *Module) Read(key string) (Map, error)`
- `func (m *Module) ReadDataFrom(conn, key string) ([]byte, error)`
- `func (m *Module) ReadData(key string) ([]byte, error)`
- `func (m *Module) WriteTo(conn string, key string, val Map, expires ...time.Duration) error`
- `func (m *Module) Write(key string, val Map, expires ...time.Duration) error`
- `func (m *Module) WriteDataTo(conn, key string, data []byte, expires ...time.Duration) error`
- `func (m *Module) WriteData(key string, data []byte, expires ...time.Duration) error`
- `func (m *Module) DeleteFrom(conn, key string) error`
- `func (m *Module) Delete(key string) error`
- `func (m *Module) SequenceOn(conn, key string, start, step int64, expires ...time.Duration) (int64, error)`
- `func (m *Module) Sequence(key string, start, step int64, expires ...time.Duration) (int64, error)`
- `func (m *Module) KeysFrom(conn string, prefixs ...string) ([]string, error)`
- `func (m *Module) Keys(prefixs ...string) ([]string, error)`
- `func (m *Module) ClearFrom(conn string, prefixs ...string) error`
- `func (m *Module) Clear(prefixs ...string) error`
- `func (d *defaultDriver) Connect(inst *Instance) (Connect, error)`
- `func (c *defaultConnection) Open() error  { return nil }`
- `func (c *defaultConnection) Close() error { return nil }`
- `func (c *defaultConnection) Read(key string) ([]byte, error)`
- `func (c *defaultConnection) Write(key string, val []byte, expire time.Duration) error`
- `func (c *defaultConnection) Exists(key string) (bool, error)`
- `func (c *defaultConnection) Delete(key string) error`
- `func (c *defaultConnection) Sequence(key string, start, step int64, expire time.Duration) (int64, error)`
- `func (c *defaultConnection) Keys(prefix string) ([]string, error)`
- `func (c *defaultConnection) Clear(prefix string) error`
- `func Read(key string) (Map, error)`
- `func ReadFrom(conn, key string) (Map, error)`
- `func ReadData(key string) ([]byte, error)`
- `func ReadDataFrom(conn, key string) ([]byte, error)`
- `func Write(key string, value Map, expires ...time.Duration) error`
- `func WriteTo(conn, key string, value Map, expires ...time.Duration) error`
- `func WriteData(key string, data []byte, expires ...time.Duration) error`
- `func WriteDataTo(conn, key string, data []byte, expires ...time.Duration) error`
- `func Delete(key string) error`
- `func DeleteFrom(conn, key string) error`
- `func Exists(key string) (bool, error)`
- `func ExistsIn(conn, key string) (bool, error)`

## 排错

- 模块未运行：确认空导入已存在
- driver 无效：确认驱动包已引入
- 配置不生效：检查配置段名是否为 `[cache]`
