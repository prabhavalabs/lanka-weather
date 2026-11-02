package regions

import (
	"container/list"
	"sync"
)

// byteCache is a least-recently-used cache of response bodies bounded by
// their total size. Keys include the data version, so a reload makes old
// entries unreachable and they age out.
type byteCache struct {
	mu       sync.Mutex
	maxBytes int
	size     int
	order    *list.List
	entries  map[string]*list.Element
}

type cacheEntry struct {
	key   string
	value []byte
}

func newByteCache(maxBytes int) *byteCache {
	return &byteCache{maxBytes: maxBytes, order: list.New(), entries: map[string]*list.Element{}}
}

func (cache *byteCache) get(key string) ([]byte, bool) {
	cache.mu.Lock()
	defer cache.mu.Unlock()
	element, ok := cache.entries[key]
	if !ok {
		return nil, false
	}
	cache.order.MoveToFront(element)
	return element.Value.(*cacheEntry).value, true
}

// cost counts an entry's bookkeeping too, so many empty bodies (tiles over
// the sea) are bounded like large ones.
func cost(entry *cacheEntry) int { return len(entry.key) + len(entry.value) + 64 }

func (cache *byteCache) put(key string, value []byte) {
	entry := &cacheEntry{key: key, value: value}
	if cost(entry) > cache.maxBytes/8 {
		return // one entry may not crowd out the rest
	}
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if element, ok := cache.entries[key]; ok {
		cache.size -= cost(element.Value.(*cacheEntry))
		element.Value = entry
		cache.order.MoveToFront(element)
	} else {
		cache.entries[key] = cache.order.PushFront(entry)
	}
	cache.size += cost(entry)
	for cache.size > cache.maxBytes {
		oldest := cache.order.Back()
		evicted := oldest.Value.(*cacheEntry)
		cache.order.Remove(oldest)
		delete(cache.entries, evicted.key)
		cache.size -= cost(evicted)
	}
}
