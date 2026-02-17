package middleware

import (
	"testing"
	"time"
)

func TestRateLimiter_Allow(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	ip := "192.168.1.1"

	for i := 0; i < 5; i++ {
		if !rl.Allow(ip) {
			t.Errorf("request %d should be allowed within burst", i+1)
		}
	}

	if rl.Allow(ip) {
		t.Error("request beyond burst should be denied")
	}
}

func TestRateLimiter_Refill(t *testing.T) {
	rl := NewRateLimiter(10, 5)

	ip := "192.168.1.1"

	for i := 0; i < 5; i++ {
		rl.Allow(ip)
	}

	if rl.Allow(ip) {
		t.Error("should be denied after burst exhausted")
	}

	time.Sleep(150 * time.Millisecond)

	if !rl.Allow(ip) {
		t.Error("should be allowed after refill")
	}
}

func TestRateLimiter_DifferentIPs(t *testing.T) {
	rl := NewRateLimiter(10, 2)

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	rl.Allow(ip1)
	rl.Allow(ip1)

	if rl.Allow(ip1) {
		t.Error("ip1 should be denied")
	}

	if !rl.Allow(ip2) {
		t.Error("ip2 should be allowed (separate bucket)")
	}
}

func TestRateLimiter_BurstCap(t *testing.T) {
	rl := NewRateLimiter(100, 5)

	ip := "192.168.1.1"

	for i := 0; i < 3; i++ {
		rl.Allow(ip)
	}

	time.Sleep(200 * time.Millisecond)

	count := 0
	for rl.Allow(ip) {
		count++
		if count > 10 {
			t.Fatal("rate limiter not respecting burst cap")
		}
	}

	if count != 5 {
		t.Errorf("expected 5 tokens after refill, got %d", count)
	}
}
