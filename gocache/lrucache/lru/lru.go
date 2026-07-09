package lru

import (
	"container/list"
)

type Cache struct {
	maxBytes int64
	nBytes   int64
	ll       *list.List
	cache    map[string]*list.Element
	//	onEvicted func(key string, value Value)
	OnEvicted func(key string, Value Value)
}

// entry is the value in the cache
type entry struct {
	key   string
	value Value
}

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

func New(maxBytes int64, onEvicted func(key string, value Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

// Add adds a value to the cache
func (c *Cache) Add(key string, value Value) {
	if e, ok := c.cache[key]; ok {
		c.ll.MoveToFront(e)
		kv := e.Value.(*entry)
		c.nBytes += int64(value.Len()) - int64(kv.value.Len())

		e.Value.(*entry).value = value
		return
	} else {
		e = c.ll.PushFront(&entry{key, value})
		c.cache[key] = e
		c.nBytes += int64(len(key)) + int64(value.Len())
	}
	// 如果使用的字节已经超过限制，这里将最少使用的元素移除
	for c.maxBytes != 0 && c.maxBytes < c.nBytes {
		c.RemoveOldest()
	}
}

// RemoveOldest 如果字节已经超出限制这里清理最老的元素
func (c *Cache) RemoveOldest() {
	// 去除末尾元素
	ele := c.ll.Back()
	if ele != nil {
		// 从链表中移除
		c.ll.Remove(ele)
		// 去除元素的值
		kv := ele.Value.(*entry)
		// 从map中删除
		delete(c.cache, kv.key)
		c.nBytes -= int64(len(kv.key)) + int64(kv.value.Len())
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Get 获取一个元素
func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// 将元素移动到链表头部
		c.ll.MoveToFront(ele)
		kv := ele.Value.(*entry)
		return kv.value, true
	}
	return
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}
