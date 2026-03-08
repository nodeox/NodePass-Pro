package middleware

import (
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisNonceStoreAddIfAbsent(t *testing.T) {
	mockRedis, err := miniredis.Run()
	if err != nil {
		t.Fatalf("start miniredis failed: %v", err)
	}
	defer mockRedis.Close()

	client := redis.NewClient(&redis.Options{Addr: mockRedis.Addr()})
	defer client.Close()

	store := NewRedisNonceStore(client, "nonce:")
	ttl := 2 * time.Second

	added, err := store.AddIfAbsent("abc", ttl)
	if err != nil {
		t.Fatalf("first AddIfAbsent returned error: %v", err)
	}
	if !added {
		t.Fatalf("first AddIfAbsent should be true")
	}

	added, err = store.AddIfAbsent("abc", ttl)
	if err != nil {
		t.Fatalf("second AddIfAbsent returned error: %v", err)
	}
	if added {
		t.Fatalf("second AddIfAbsent should be false")
	}

	mockRedis.FastForward(3 * time.Second)
	added, err = store.AddIfAbsent("abc", ttl)
	if err != nil {
		t.Fatalf("third AddIfAbsent returned error: %v", err)
	}
	if !added {
		t.Fatalf("third AddIfAbsent should be true after ttl expired")
	}
}
