package doh

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

var testProviders = []string{
	"https://cloudflare-dns.com/dns-query",
	"https://1.12.12.12/dns-query",
	"https://dns.google/resolve",
	"https://9.9.9.9:5053/dns-query",
}

func TestNewSingle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c := New(WithProviders([]string{"https://cloudflare-dns.com/dns-query"}))
	defer c.Close()

	rsp, err := c.Query(ctx, "example.com", TypeA)
	if err != nil {
		t.Fatalf("Query() returned an unexpected error: %v", err)
	}
	if len(rsp.Answer) == 0 {
		t.Errorf("Expected at least one answer, got 0")
	}
}

func TestNew(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := New()
	defer c.Close()

	_, err := c.Query(ctx, "example", TypeA) // Invalid TLD
	if err == nil {
		t.Errorf("Expected an error for an invalid domain, but got nil")
	}

	c = New(WithProviders(testProviders))
	for i := 0; i < 10; i++ { // Reduced loop for faster testing
		for _, v := range []Type{TypeA, TypeMX} {
			rsp, err := c.Query(ctx, "example.com", v)
			if err != nil {
				t.Fatalf("Query() returned an unexpected error: %v", err)
			}
			if len(rsp.Answer) == 0 {
				t.Errorf("Expected at least one answer for type %s, got 0", v)
			}
		}
	}
}

func TestEnableCache(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := New()
	defer c.Close()

	c.EnableCache(true)
	rsp, err := c.Query(ctx, "example.com", TypeA)
	if err != nil {
		t.Fatalf("Query() returned an unexpected error: %v", err)
	}
	if len(rsp.Answer) == 0 {
		t.Fatal("Expected at least one answer, got 0")
	}
	ttl := rsp.Answer[0].TTL

	time.Sleep(1 * time.Second)
	rsp, err = c.Query(ctx, "example.com", TypeA)
	if err != nil {
		t.Fatalf("Query() from cache returned an unexpected error: %v", err)
	}
	if len(rsp.Answer) == 0 {
		t.Fatal("Expected at least one answer from cache, got 0")
	}
	if rsp.Answer[0].TTL != ttl {
		t.Errorf("Expected TTL from cache to be %d, but got %d", ttl, rsp.Answer[0].TTL)
	}

	c.EnableCache(false)
	rsp, err = c.Query(ctx, "example.com", TypeA)
	if err != nil {
		t.Fatalf("Query() with cache disabled returned an unexpected error: %v", err)
	}
	if len(rsp.Answer) == 0 {
		t.Fatal("Expected at least one answer with cache disabled, got 0")
	}
	ttl = rsp.Answer[0].TTL

	time.Sleep(1 * time.Second)
	rsp, err = c.Query(ctx, "example.com", TypeA)
	if err != nil {
		t.Fatalf("Second query with cache disabled returned an unexpected error: %v", err)
	}
	if len(rsp.Answer) == 0 {
		t.Fatal("Expected at least one answer on second query with cache disabled, got 0")
	}
	if rsp.Answer[0].TTL == ttl {
		t.Errorf("Expected TTL to change, but it remained %d", ttl)
	}
}

func TestConcurrentQuery(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	c := New(WithProviders(testProviders))
	defer c.Close()

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			rsp, err := c.Query(ctx, "example.com", TypeA)
			if err != nil {
				t.Errorf("Query() returned an unexpected error: %v", err)
				return
			}
			if len(rsp.Answer) == 0 {
				t.Errorf("Expected at least one answer, got 0")
			}
		}()
	}

	wg.Wait()
}

func TestWithOptions(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test with a very short timeout that should fail
	c := New(WithTimeout(1 * time.Millisecond))
	defer c.Close()

	_, err := c.Query(ctx, "example.com", TypeA)
	if err == nil {
		t.Fatal("Expected an error for a short timeout, but got nil")
	}

	if !errors.Is(err, context.DeadlineExceeded) {
		t.Errorf("Expected a context.DeadlineExceeded error, but got: %v", err)
	}
}
