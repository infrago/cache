# cache

`cache` 是 infrago 的**模块**。

## 包定位

- 类型：模块
- 作用：统一缓存模块，负责缓存读写与多后端抽象。

## 主要功能

- 对上提供统一模块接口
- 对下通过驱动接口接入具体后端
- 支持按配置切换驱动实现
- 支持命中/未命中/错误等模块级和实例级基础统计
- 支持单个或批量自动序列号
- 支持统一 key 拼接，避免业务里散落手写 key 规则

## 快速接入

```go
import _ "github.com/infrago/cache"
```

```toml
[cache]
driver = "default"
prefix = "app:"
expire = "10m"
allow_clear_all = false
```

```go
key := cache.Key("user", 123, "profile") // user:123:profile
```

## 驱动实现接口列表

以下接口由驱动实现（来自模块 `driver.go`）：

### Driver

- `Connect(*Instance) (Connect, error)`

### Connect

- `Open() error`
- `Close() error`
- `Read(string) ([]byte, error)`
- `Write(key string, val []byte, expire time.Duration) error`
- `Exists(key string) (bool, error)`
- `Delete(key string) error`
- `Sequence(key string, start, step int64, expire time.Duration) (int64, error)`
- `Keys(prefix string) ([]string, error)`
- `Clear(prefix string) error`

### ManyConnect

`ManyConnect` 是可选接口。驱动实现后，`SequenceMany` 会走驱动原生批量能力；未实现时模块会自动循环调用 `Sequence`。

- `SequenceMany(key string, start, step, count int64, expire time.Duration) ([]int64, error)`

## 全局配置项（所有配置键）

配置段：`[cache]` 或 `[cache.xxx]`

- `driver`：驱动名，默认 `default`
- `weight`：分布权重，默认 `1`；小于 `0` 表示打开实例但不参与默认 key 分布
- `prefix`：真实缓存 key 前缀
- `codec`：Map 编解码器，默认 `json`
- `expire` / `ttl`：默认过期时间，支持 `"10s"` 这类 duration 字符串，也支持数字秒
- `allow_clear_all` / `clear_all`：是否允许 `Clear()` 空前缀全清，默认 `false`
- `setting`：驱动专用配置

## 管理操作

- `Keys` / `Clear` 是管理操作，不建议在高频业务路径中调用。
- `Clear("")` / `Clear()` 默认会返回 `ErrUnsafeClear`，避免误删整个实例。
- 确认需要全清时使用 `ClearAll()` / `ClearAllFrom(conn)`，或者配置 `allow_clear_all=true` 兼容旧调用。
- Redis 驱动内部使用 `SCAN` 避免 `KEYS` 阻塞，但大范围扫描仍然有成本。

## 说明

- `setting` 一般用于向具体驱动透传专用参数
- 多实例配置请参考模块源码中的 Config/configure 处理逻辑
- `Stats()` 返回当前进程内基础计数，并在 `Instances` 中包含实例级统计
- `StatsFrom(conn)` 返回指定实例统计；`ResetStats()` 清空模块和实例计数
