package cache

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"
)

var (
	FileCachePath           = "runtime/cache" // 缓存目录
	FileCacheFileSuffix     = ".gob"                        // 缓存文件后缀
	FileCacheDirectoryLevel = 1                             // 缓存目录层级
	FileCacheExpire         time.Duration                   // 缓存过期时间
)

type FileItem struct {
	Val        interface{} // 缓存内容
	LastAccess time.Time   // 最后访问时间
	Expire     time.Time   // 缓存有效期
}

type FileCache struct {
	CachePath      string // 缓存目录
	FileSuffix     string // 缓存文件后缀
	DirectoryLevel int    // 缓存目录层级
	CacheExpire    int    // 缓存过期时间
}

// 返回新的文件缓存驱动
func NewFileCache() Cache {
	return &FileCache{}
}

// 获取一个缓存
func (fc *FileCache) Get(key string) interface{} {
	fileData, err := FileGetContents(fc.getCacheFileName(key))
	if err != nil {
		return ""
	}
	var to FileItem
	GobDecode(fileData, &to)
	if to.Expire.Before(time.Now()) {
		return ""
	}
	return to.Val
}

// 获取多个缓存
func (fc *FileCache) GetMulti(keys []string) []interface{} {
	var rc []interface{}
	for _, key := range keys {
		rc = append(rc, fc.Get(key))
	}
	return rc
}

// 设置一个缓存
func (fc *FileCache) Put(key string, val interface{}, timeout time.Duration) error {
	gob.Register(val)
	item := FileItem{Val: val}

	if timeout == FileCacheExpire {
		item.Expire = time.Now().Add((86400 * 365 * 10) * time.Second) // 十年
	} else {
		item.Expire = time.Now().Add(timeout)
	}
	item.LastAccess = time.Now()
	data, err := GobEncode(item)
	if err != nil {
		return err
	}
	return FilePutContents(fc.getCacheFileName(key), data)
}

// 删除一个缓存
func (fc *FileCache) Delete(key string) error {
	filename := fc.getCacheFileName(key)
	if ok, _ := exists(filename); ok {
		return os.Remove(filename)
	}
	return nil
}

// 自增一个值
func (fc *FileCache) Incr(key string) error {
	val := fc.Get(key)
	var incr int
	if reflect.TypeOf(val).Name() != "int" {
		incr = 0
	} else {
		incr = val.(int) + 1
	}
	fc.Put(key, incr, FileCacheExpire)
	return nil
}

// 自减一个值
func (fc *FileCache) Decr(key string) error {
	val := fc.Get(key)
	var decr int
	if reflect.TypeOf(val).Name() != "int" || val.(int)-1 <= 0 {
		decr = 0
	} else {
		decr = val.(int) - 1
	}
	fc.Put(key, decr, FileCacheExpire)
	return nil
}

// 检查缓存是否存在
func (fc *FileCache) IsExist(key string) bool {
	ret, _ := exists(fc.getCacheFileName(key))
	return ret
}

// 清除所有缓存
func (fc *FileCache) ClearAll() error {
	return os.RemoveAll(fc.CachePath)
}

// 获取缓存文件名
func (fc *FileCache) getCacheFileName(key string) string {
	m := md5.New()
	io.WriteString(m, key)
	keyMd5 := hex.EncodeToString(m.Sum(nil))
	cachePath := fc.CachePath
	switch fc.DirectoryLevel {
	case 2:
		cachePath = filepath.Join(cachePath, keyMd5[0:2], keyMd5[2:4])
	case 1:
		cachePath = filepath.Join(cachePath, keyMd5[0:2])
	}

	if ok, _ := exists(cachePath); !ok {
		_ = os.MkdirAll(cachePath, os.ModePerm)
	}
	return filepath.Join(cachePath, fmt.Sprintf("%s%s", keyMd5, fc.FileSuffix))
}

// 启动
func (fc *FileCache) StartAndGC(config string) error {
	var cfg map[string]string
	json.Unmarshal([]byte(config), &cfg)
	if _, ok := cfg["CachePath"]; !ok {
		cfg["CachePath"] = FileCachePath
	}
	if _, ok := cfg["FileSuffix"]; !ok {
		cfg["FileSuffix"] = FileCacheFileSuffix
	}
	if _, ok := cfg["DirectoryLevel"]; !ok {
		cfg["DirectoryLevel"] = strconv.Itoa(FileCacheDirectoryLevel)
	}
	if _, ok := cfg["CacheExpire"]; !ok {
		cfg["CacheExpire"] = strconv.FormatInt(int64(FileCacheExpire.Seconds()), 10)
	}
	fc.CachePath = cfg["CachePath"]
	fc.FileSuffix = cfg["FileSuffix"]
	fc.DirectoryLevel, _ = strconv.Atoi(cfg["DirectoryLevel"])
	fc.CacheExpire, _ = strconv.Atoi(cfg["CacheExpire"])
	if ok, _ := exists(fc.CachePath); !ok {
		_ = os.MkdirAll(fc.CachePath, os.ModePerm)
	}
	return nil
}

// 检查路径是否存在
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// 获取文件内容
func FileGetContents(filename string) (data []byte, e error) {
	return ioutil.ReadFile(filename)
}

// 写文件，不存在创建
func FilePutContents(filename string, content []byte) error {
	return ioutil.WriteFile(filename, content, os.ModePerm)
}

// GobEncode编码
func GobEncode(data interface{}) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(data)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), err
}

// GobDecode解码
func GobDecode(data []byte, to *FileItem) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(&to)
}

func init() {
	Register("file", NewFileCache)
}
