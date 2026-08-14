package db

import (
	"container/list"

	"github.com/belldb/internal/storage"
)

type cacheEntry struct {
	idx    int
	points []storage.Point
}

type Cache struct {
	capacity int
	items    map[int]*list.Element
	lru      *list.List
}

func NewChunkCache(capacity int) *Cache {
	return &Cache{
		capacity: capacity,
		items:    make(map[int]*list.Element),
		lru:      list.New(),
	}
}

func (c *Cache) Get(index int) ([]storage.Point, bool) {
	elem, ok := c.items[index]
	if !ok {
		return nil, false
	}

	c.lru.MoveToFront(elem)

	return elem.Value.(*cacheEntry).points, true
}

func (c *Cache) Put(idx int, points []storage.Point) {
	if elem, ok := c.items[idx]; ok {
		elem.Value.(*cacheEntry).points = points
		c.lru.MoveToFront(elem)
		return
	}

	elem := c.lru.PushFront(&cacheEntry{
		idx:    idx,
		points: points,
	})

	c.items[idx] = elem

	if c.lru.Len() > c.capacity {
		oldest := c.lru.Back()

		entry := oldest.Value.(*cacheEntry)

		delete(c.items, entry.idx)
		c.lru.Remove(oldest)
	}
}
