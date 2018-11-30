package cache

import (
	"fmt"
	"time"
)

// 缓存接口
type Cache interface {
	// 获取缓存
	Get(key string) interface{}
	// 获取多个缓存
	GetMulti(keys []string) []interface{}
	// 设置缓存和有效期
	Put(key string, val interface{}, timeout time.Duration) error
	// 删除一个缓存
	Delete(key string) error
	// 自增一个值
	Incr(key string) error
	// 自减一个值
	Decr(key string) error
	// 检查key是否存在
	IsExist(key string) bool
	// 清除所有缓存
	ClearAll() error
	// 启动并回收
	StartAndGC(config string) error
}

// 实例是一个函数，创建一个新的缓存实例
type Instance func() Cache

// 所有的缓存适配器
var adapters = make(map[string]Instance)

// 注册一个新的适配器
func Register(name string, adapter Instance) {
	if adapter == nil {
		panic("cache: 注册适配器不存在")
	}
	if _, ok := adapters[name]; ok {
		panic("cache: " + name + "适配器重复注册")
	}
	adapters[name] = adapter
}

// 通过适配器名称创建一个新的缓存驱动，配置通过json字符串格式传入，并启动gc
func NewCache(adapterName, config string) (adapter Cache, err error) {
	instanceFunc, ok := adapters[adapterName]
	if !ok {
		err = fmt.Errorf("cache: 未知适配器名 %q", adapterName)
		return
	}
	adapter = instanceFunc()
	err = adapter.StartAndGC(config)
	if err != nil {
		adapter = nil
	}
	return
}