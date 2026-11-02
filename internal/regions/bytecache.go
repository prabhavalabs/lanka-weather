package regions

// ByteCache is the bounded response cache regions keep their tiles in, for
// other tiled layers.
type ByteCache struct{ cache *byteCache }

// NewByteCache returns an empty cache of at most maxBytes.
func NewByteCache(maxBytes int) *ByteCache { return &ByteCache{cache: newByteCache(maxBytes)} }

// Get returns a cached body.
func (cache *ByteCache) Get(key string) ([]byte, bool) { return cache.cache.get(key) }

// Put stores a body, evicting the least recently used.
func (cache *ByteCache) Put(key string, value []byte) { cache.cache.put(key, value) }
