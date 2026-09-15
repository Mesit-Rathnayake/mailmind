package cache

import (
	"testing"
	"time"
)

func TestMemoryCache(t *testing.T) {
	c := NewMemoryCache()

	// 1. Basic Set & Get
	c.Set("test_key", "hello_world", 100*time.Millisecond)
	val, ok := c.Get("test_key")
	if !ok || val != "hello_world" {
		t.Fatalf("expected hello_world, got %v (ok: %v)", val, ok)
	}

	// 2. Expiration
	time.Sleep(150 * time.Millisecond)
	_, ok = c.Get("test_key")
	if ok {
		t.Fatalf("expected key to be expired")
	}

	// 3. Clear
	c.Set("key1", "val1", 1*time.Minute)
	c.Set("key2", "val2", 1*time.Minute)
	c.Clear()

	if _, ok := c.Get("key1"); ok {
		t.Fatalf("expected key1 to be cleared")
	}
	if _, ok := c.Get("key2"); ok {
		t.Fatalf("expected key2 to be cleared")
	}
}
