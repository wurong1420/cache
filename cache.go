package cache

import "time"

const (
	NoExpiration = time.Duration(-1)
	DefaultExpiration = time.Duration(30)*time.Second
	DefaultSegementSize = 16
)

type Cache interface {
	SetD(key string, value interface{})
	AddD(key string, value interface{})
	Set(key string, value interface{}, d time.Duration)
	Add(key string, value interface{}, d time.Duration)
	Get(key string) interface{}
	Size() int
}

func fnv32(key string) uint32 {
	hash := uint32(2166136261)
	const prime32 = uint32(16777619)
	for i := 0; i < len(key); i++ {
		hash *= prime32
		hash ^= uint32(key[i])
	}
	return hash
}

func NewCacheD() Cache {
	return NewCache(DefaultSegementSize)
}

func NewCache(n int) Cache {
	segments := make([]*segment, 0)
	for i :=0; i < n; i++ {
	    segments = append(segments, newSegment(1*time.Second))
	}
	return &cache{segements: segments}
}

type cache struct {
	segements []*segment
}

func (c cache) SetD(key string, value interface{}) {
	c.Set(key, value, DefaultExpiration)
}

func (c cache) AddD(key string, value interface{}) {
	c.Add(key, value, DefaultExpiration)
}

func (c cache) Set(key string, value interface{}, d time.Duration) {
	hash := int(fnv32(key)) % len(c.segements)
	segment := c.segements[hash]
	segment.set(key, value, d)
}

func (c cache) Add(key string, value interface{}, d time.Duration) {
	hash := int(fnv32(key)) % len(c.segements)
	segment := c.segements[hash]
	segment.add(key, value, d)
}

func (c cache) Get(key string) interface{} {
	hash := int(fnv32(key)) % len(c.segements)
	segment := c.segements[hash]
	return segment.get(key)
}

func (c cache) Size() int {
	size := 0
	for _, segement := range c.segements {
		size += segement.size()
	}
	return size
}