package regions

import (
	"reflect"
	"testing"
)

func TestTileLevels(t *testing.T) {
	if got := TileLevels(0); !reflect.DeepEqual(got, []string{"province"}) {
		t.Fatalf("TileLevels(0) = %v", got)
	}
	if got := TileLevels(TileMinZoom[LevelGN]); !reflect.DeepEqual(got, []string{"province", "district", "ds", "gn"}) {
		t.Fatalf("TileLevels at the GN zoom = %v", got)
	}
}

func TestByteCacheEvictsLeastRecentlyUsed(t *testing.T) {
	cache := newByteCache(1000)
	value := make([]byte, 50)
	cache.put("a", value)
	cache.put("b", value)
	if _, ok := cache.get("a"); !ok {
		t.Fatal("a missing")
	}
	for i := range 20 {
		cache.put(string(rune('c'+i)), value)
	}
	if _, ok := cache.get("b"); ok {
		t.Fatal("b should have been evicted")
	}
	if cache.size > cache.maxBytes {
		t.Fatalf("cache size %d exceeds %d", cache.size, cache.maxBytes)
	}
	cache.put("huge", make([]byte, 900))
	if _, ok := cache.get("huge"); ok {
		t.Fatal("an entry larger than an eighth of the cache should not be stored")
	}
}
