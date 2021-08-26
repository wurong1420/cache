# cache
How to use it
```go
  cache := NewCache(16)
	cache.Set("hello", "world", time.Duration(5)*time.Second)
```
