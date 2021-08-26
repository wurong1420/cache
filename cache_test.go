package cache

import (
	"fmt"
	"testing"
	"time"
)

var (
	c Cache
)

func TestMain(m *testing.M) {
	c = NewCache(16)
	for i :=0; i < 1000000; i++ {
		c.Set(fmt.Sprintf("hello_%d", i), "world", time.Duration(3000000)*time.Second)
	}

	m.Run()
}

func TestNewCache(t *testing.T) {
	cache := NewCache(16)
	cache.Set("hello", "world", time.Duration(5)*time.Second)

	time.Sleep(3*time.Second)
	value := cache.Get("hello")
	fmt.Println("缓存结果：", value)

	time.Sleep(5*time.Second)
	value = cache.Get("hello")
	fmt.Println("再查一次：", value)
}

func TestCache_Get(t *testing.T) {
	value := c.Get(fmt.Sprintf("hello_%d", 800000))
	fmt.Println(value)
}

func BenchmarkCache_Set(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c.Set(fmt.Sprintf("hello_%d", i), "world", time.Duration(30)*time.Second)
	}
	fmt.Println(c.Size())
}

func BenchmarkCache_Get(b *testing.B) {
	for i := 0; i < b.N; i++ {
		c.Get(fmt.Sprintf("hello_%d", 800000))
	}
}