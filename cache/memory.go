package cache

import (
	"encoding/json"
	"errors"
	"sync"
	"time"
)

var (
	// 默认回收过期缓存时间，默认一分钟
	DefaultEvery int = 60
)

// 缓存item结构
type MemoryItem struct {
	val       interface{}
	createdAt time.Time
	ttr       time.Duration
}

// 是否过期
func (m *MemoryItem) isExpire() bool {
	if m.ttr == 0 {
		return false
	}
	return time.Now().Sub(m.createdAt) > m.ttr
}

// 缓冲驱动结构
type MemoryCache struct {
	sync.RWMutex //读写锁
	duration     time.Duration
	items        map[string]*MemoryItem
	Every        int
}

// 返回新的缓存
func NewMemoryCache() Cache {
	cache := MemoryCache{items: make(map[string]*MemoryItem)}
	return &cache
}

// 获取一个缓存
func (bc *MemoryCache) Get(name string) interface{} {
	bc.RLock()
	defer bc.RUnlock()
	if item, ok := bc.items[name]; ok {
		if item.isExpire() {
			return nil
		}
		return item.val
	}
	return nil
}

// 获取多个缓存
func (bc *MemoryCache) GetMulti(names []string) []interface{} {
	var rc []interface{}
	for _, name := range names {
		rc = append(rc, bc.Get(name))
	}
	return rc
}

// 设置一个缓存
// 如果ttr = 0 永久缓存
func (bc *MemoryCache) Put(name string, value interface{}, ttr time.Duration) error {
	bc.Lock()
	defer bc.Unlock()
	bc.items[name] = &MemoryItem{
		val:       value,
		createdAt: time.Now(),
		ttr:       ttr,
	}
	return nil
}

// 删除一个缓存
func (bc *MemoryCache) Delete(name string) error {
	bc.Lock()
	defer bc.Unlock()
	if _, ok := bc.items[name]; !ok {
		return errors.New("key: + " + name + "不存在")
	}
	delete(bc.items, name)
	if _, ok := bc.items[name]; ok {
		return errors.New("key: + " + name + "删除出错")
	}
	return nil
}

// 自增 支持int int32 int66 uint uint32 unit64
func (bc *MemoryCache) Incr(key string) error {
	bc.RLock()
	defer bc.RUnlock()
	item, ok := bc.items[key]
	if !ok {
		return errors.New("key:" + key + "不存在")
	}
	switch item.val.(type) {
	case int:
		item.val = item.val.(int) + 1
	case int32:
		item.val = item.val.(int32) + 1
	case int64:
		item.val = item.val.(int64) + 1
	case uint:
		item.val = item.val.(uint) + 1
	case uint32:
		item.val = item.val.(uint32) + 1
	case uint64:
		item.val = item.val.(uint64) + 1
	default:
		return errors.New("key:" + key + "的值不是 (u)int (u)int32 (u)int64 类型")
	}
	return nil
}

func (bc *MemoryCache) Decr(key string) error {
	bc.RLock()
	defer bc.RUnlock()
	item, ok := bc.items[key]
	if !ok {
		return errors.New("key:" + key + "不存在")
	}
	switch item.val.(type) {
	case int:
		item.val = item.val.(int) - 1
	case int64:
		item.val = item.val.(int64) - 1
	case int32:
		item.val = item.val.(int32) - 1
	case uint:
		if item.val.(uint) > 0 {
			item.val = item.val.(uint) - 1
		} else {
			return errors.New("key:" + key + "的值不能小于0")
		}
	case uint32:
		if item.val.(uint32) > 0 {
			item.val = item.val.(uint32) - 1
		} else {
			return errors.New("key:" + key + "的值不能小于0")
		}
	case uint64:
		if item.val.(uint64) > 0 {
			item.val = item.val.(uint64) - 1
		} else {
			return errors.New("key:" + key + "的值不能小于0")
		}
	default:
		return errors.New("key:" + key + "的值不是 int int32 int64 类型")
	}
	return nil
}

// 检查是否存在缓存
func (bc *MemoryCache) IsExist(name string) bool {
	bc.RLock()
	defer bc.RUnlock()
	if v, ok := bc.items[name]; ok {
		return !v.isExpire()
	}
	return false
}

// 清除所有缓存
func (bc *MemoryCache) ClearAll() error {
	bc.Lock()
	defer bc.Unlock()
	bc.items = make(map[string]*MemoryItem)
	return nil
}

// 启动
func (bc *MemoryCache) StartAndGC(config string) error {
	var cf map[string]int
	json.Unmarshal([]byte(config), &cf)
	if _, ok := cf["interval"]; !ok {
		cf = make(map[string]int)
		cf["interval"] = DefaultEvery
	}
	duration := time.Duration(cf["interval"]) * time.Second
	bc.Every = cf["interval"]
	bc.duration = duration
	go bc.vacuum()
	return nil
}

// 自动gc
func (bc *MemoryCache) vacuum() {
	bc.RLock()
	every := bc.Every
	bc.RUnlock()
	if every < 1 {
		return
	}
	for {
		<-time.After(bc.duration)
		if bc.items == nil {
			return
		}
		if keys := bc.expiredKeys(); len(keys) != 0 {
			bc.clearItems(keys)
		}
	}
}

// 返回所有在有效期内的key
func (bc *MemoryCache) expiredKeys() (keys []string) {
	bc.RLock()
	defer bc.RUnlock()
	for key, item := range bc.items {
		if item.isExpire() {
			keys = append(keys, key)
		}
	}
	return
}

// 清除指定多个key的缓存
func (bc *MemoryCache) clearItems(keys []string) {
	bc.Lock()
	defer bc.Unlock()
	for _, key := range keys {
		delete(bc.items, key)
	}
}

func init() {
	Register("memory", NewMemoryCache)
}
