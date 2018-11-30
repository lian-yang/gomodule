package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"strconv"
	"strings"
	"time"
)

var (
	DefaultKey = "redisCache"
)

type RedisCache struct {
	p        *redis.Pool
	db       int
	dsn      string
	key      string
	password string
	maxIdle  int
}

func (rc *RedisCache) Get(key string) interface{} {
	if v, err := rc.do("GET", key); err == nil {
		return v
	}
	return nil
}

func (rc *RedisCache) GetMulti(keys []string) []interface{} {
	c := rc.p.Get()
	defer c.Close()
	var args []interface{}
	for _, key := range keys {
		args = append(args, rc.associate(key))
	}
	values, err := redis.Values(c.Do("MGET", args...))
	if err != nil {
		return nil
	}
	return values
}

func (rc *RedisCache) Put(key string, val interface{}, timeout time.Duration) error {
	_, err := rc.do("SETEX", key, int64(timeout/time.Second), val)
	return err
}

func (rc *RedisCache) Delete(key string) error {
	_, err := rc.do("DEL", key)
	return err
}

func (rc *RedisCache) Incr(key string) error {
	_, err := redis.Bool(rc.do("INCRBY", key, 1))
	return err
}

func (rc *RedisCache) Decr(key string) error {
	_, err := redis.Bool(rc.do("INCRBY", key, -1))
	return err
}

func (rc *RedisCache) IsExist(key string) bool {
	v, err := redis.Bool(rc.do("EXISTS", key))
	if err != nil {
		return false
	}
	return v
}

func (rc *RedisCache) ClearAll() error {
	c := rc.p.Get()
	defer c.Close()
	cachedKeys, err := redis.Strings(c.Do("KEYS", rc.key+":*"))
	if err != nil {
		return err
	}
	for _, str := range cachedKeys {
		if _, err = c.Do("DEL", str); err != nil {
			return err
		}
	}
	return err
}

func (rc *RedisCache) StartAndGC(config string) error {
	var cfg map[string]string
	json.Unmarshal([]byte(config), &cfg)
	if _, ok := cfg["key"]; !ok {
		cfg["key"] = DefaultKey
	}
	if _, ok := cfg["dsn"]; !ok {
		return errors.New("redis链接不存在")
	}
	cfg["dsn"] = strings.Replace(cfg["dsn"], "redis://", "", 1)
	if i := strings.Index(cfg["dsn"], "@"); i > -1 {
		cfg["password"] = cfg["dsn"][0:i]
		cfg["dsn"] = cfg["dsn"][i+1:]
	}
	if _, ok := cfg["db"]; !ok {
		cfg["db"] = "0"
	}
	if _, ok := cfg["password"]; !ok {
		cfg["password"] = ""
	}
	if _, ok := cfg["maxIdle"]; !ok {
		cfg["maxIdle"] = "3"
	}
	rc.key = cfg["key"]
	rc.dsn = cfg["dsn"]
	rc.db, _ = strconv.Atoi(cfg["db"])
	rc.password = cfg["password"]
	rc.maxIdle, _ = strconv.Atoi(cfg["maxIdle"])
	rc.connect()
	c := rc.p.Get()
	defer c.Close()
	return c.Err()
}

func (rc *RedisCache) connect() {
	dialFunc := func() (c redis.Conn, err error) {
		c, err = redis.Dial("tcp", rc.dsn)
		if err != nil {
			return nil, err
		}

		if rc.password != "" {
			if _, err := c.Do("AUTH", rc.password); err != nil {
				c.Close()
				return nil, err
			}
		}

		_, selecterr := c.Do("SELECT", rc.db)
		if selecterr != nil {
			c.Close()
			return nil, selecterr
		}
		return
	}
	// initialize a new pool
	rc.p = &redis.Pool{
		MaxIdle:     rc.maxIdle,
		IdleTimeout: 180 * time.Second,
		Dial:        dialFunc,
	}
}

func (rc *RedisCache) do(commandName string, args ...interface{}) (reply interface{}, err error) {
	if len(args) < 1 {
		return nil, errors.New("missing required arguments")
	}
	args[0] = rc.associate(args[0])
	c := rc.p.Get()
	defer c.Close()
	return c.Do(commandName, args...)
}

func (rc *RedisCache) associate(originKey interface{}) string {
	return fmt.Sprintf("%s:%s", rc.key, originKey)
}

func NewRedisCache() Cache {
	return &RedisCache{key: DefaultKey}
}

func init() {
	Register("redis", NewRedisCache)
}
