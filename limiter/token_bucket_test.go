package limiter

import (
	"log"
	"sync"
	"testing"
	"time"
)

func TestTokenBucket(t *testing.T) {
	var wg sync.WaitGroup
	var lr TokenBucket
	lr.Set(3, 5) // 每秒token放入速率  桶的容量

	for i := 0; i < 10; i++ {
		wg.Add(1)

		log.Println("创建请求:", i)
		go func(i int) {
			if lr.Allow() {
				log.Println("响应请求:", i)
			} else {
				log.Println("拒绝请求:", i)
			}
			wg.Done()
		}(i)

		time.Sleep(200 * time.Millisecond)
	}
	wg.Wait()
}