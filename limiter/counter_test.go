package limiter

import (
	"log"
	"sync"
	"testing"
	"time"
)

func TestCounter(t *testing.T) {
	var wg sync.WaitGroup
	var lr Counter
	lr.Set(3, time.Second) // 1s内最多请求3次

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