// Package doh provides a simple and flexible Go client library for DNS over HTTPS (DoH).
package doh

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Option is a function that configures a DoH client.
type Option func(*DoH)

// DoH is a DNS-over-HTTPS client.
type DoH struct {
	urls       []string
	cache      *cache
	stats      map[int][]interface{}
	stopc      chan bool
	httpClient *http.Client
	sync.RWMutex
}

// WithProviders sets the list of DoH server URLs.
func WithProviders(urls []string) Option {
	return func(d *DoH) {
		d.urls = urls
	}
}

// WithHTTPClient allows providing a custom http.Client.
func WithHTTPClient(client *http.Client) Option {
	return func(d *DoH) {
		d.httpClient = client
	}
}

// WithTransport allows providing a custom http.Transport.
func WithTransport(transport http.RoundTripper) Option {
	return func(d *DoH) {
		d.httpClient.Transport = transport
	}
}

// WithDialContext allows providing a custom dial context function.
func WithDialContext(dialer func(ctx context.Context, network, addr string) (net.Conn, error)) Option {
	return func(d *DoH) {
		transport, ok := d.httpClient.Transport.(*http.Transport)
		if !ok {
			dialer := &net.Dialer{
				Timeout:   3 * time.Second,
				KeepAlive: 60 * time.Second,
			}
			transport = &http.Transport{
				DialContext:         dialer.DialContext,
				TLSHandshakeTimeout: 3 * time.Second,
				DisableKeepAlives:   false,
				MaxIdleConns:        256,
				MaxIdleConnsPerHost: 256,
			}
		}
		transport.DialContext = dialer
		d.httpClient.Transport = transport
	}
}

// WithTimeout sets the total timeout for the HTTP client.
func WithTimeout(timeout time.Duration) Option {
	return func(d *DoH) {
		d.httpClient.Timeout = timeout
	}
}

// New returns a new DoH client, configured with the provided options.
func New(opts ...Option) *DoH {
	defaultClient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   3 * time.Second,
				KeepAlive: 60 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout: 3 * time.Second,
			DisableKeepAlives:   false,
			MaxIdleConns:        256,
			MaxIdleConnsPerHost: 256,
		},
	}

	c := &DoH{
		urls:       nil, // Will be set by WithProviders or default
		cache:      nil,
		stats:      make(map[int][]interface{}),
		stopc:      make(chan bool),
		httpClient: defaultClient,
	}

	// Apply all the options provided by the user
	for _, opt := range opts {
		opt(c)
	}

	// If no providers were set, use the default list.
	if len(c.urls) == 0 {
		c.urls = []string{
			"https://9.9.9.9:5053/dns-query", // Quad9
			"https://1.1.1.1/dns-query",      // Cloudflare
			"https://8.8.8.8/resolve",        // Google
			"https://1.12.12.12/dns-query",   // DNSPod
		}
	}

	go func() {
		t := time.NewTicker(5 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-c.stopc:
				return
			case <-t.C:
				c.Lock()
				for k := range c.stats {
					delete(c.stats, k)
				}
				c.Unlock()
			}
		}
	}()

	return c
}

// EnableCache enable query cache
func (c *DoH) EnableCache(cache bool) *DoH {
	if cache {
		c.cache = newCache()
	} else {
		c.cache = nil
	}
	return c
}

// Close close doh client
func (c *DoH) Close() {
	c.stopc <- true
	if c.cache != nil {
		c.cache.Close()
	}
}

// Query do DoH query
func (c *DoH) Query(ctx context.Context, d Domain, t Type, s ...ECS) (*Response, error) {
	urlsToQuery := make(map[int]string)

	c.RLock()
	if len(c.stats) > 0 {
		minIndex := -1
		minRate := 101.0
		for k, v := range c.stats {
			rate := v[2].(float64)
			if rate < minRate {
				minRate = rate
				minIndex = k
			}
		}
		if minIndex != -1 {
			urlsToQuery[minIndex] = c.urls[minIndex]
		}
	}

	if len(urlsToQuery) == 0 {
		for i, u := range c.urls {
			urlsToQuery[i] = u
		}
	}
	c.RUnlock()

	return c.fastQuery(ctx, urlsToQuery, d, t, s...)
}

// fastQuery do query and returns the fastest result
func (c *DoH) fastQuery(ctx context.Context,
	urls map[int]string, d Domain, t Type, s ...ECS) (*Response, error) {
	if c.cache != nil {
		if resp, ok := c.checkCache(d, t, s...); ok {
			return resp, nil
		}
	}

	ctxs, cancels := context.WithCancel(ctx)
	defer cancels()

	r := make(chan interface{})
	for originalIndex, u := range urls {
		go c.goQuery(ctxs, originalIndex, u, d, t, r, s...)
	}

	resp, err := c.collectResponses(r, len(urls))
	if err == nil && c.cache != nil {
		c.updateCache(d, t, resp, s...)
	}

	return resp, err
}

func (c *DoH) checkCache(d Domain, t Type, s ...ECS) (*Response, bool) {
	var ss string
	if len(s) > 0 && s[0] != "" {
		ss = strings.TrimSpace(string(s[0]))
	}
	hasher := sha1.New()
	hasher.Write([]byte(string(d) + string(t) + ss))
	cacheKey := hex.EncodeToString(hasher.Sum(nil))
	v := c.cache.Get(cacheKey)
	if v != nil {
		return v.(*Response), true
	}
	return nil, false
}

func (c *DoH) goQuery(ctx context.Context, k int, u string, d Domain, t Type, r chan<- interface{}, s ...ECS) {
	rsp, err := c.query(ctx, u, d, t, s...)
	c.Lock()
	if _, ok := c.stats[k]; !ok {
		c.stats[k] = []interface{}{0, 0, 100.0}
	}
	c.stats[k][1] = c.stats[k][1].(int) + 1
	if err != nil {
		c.stats[k][0] = c.stats[k][0].(int) + 1
	}
	c.stats[k][2] = float64(c.stats[k][0].(int)) / float64(c.stats[k][1].(int)) * 100
	c.Unlock()

	if err == nil {
		r <- rsp
	} else {
		r <- err
	}
}

func (c *DoH) collectResponses(r chan interface{}, totalUrls int) (*Response, error) {
	var firstError error
	total := 0
	for v := range r {
		total++
		if resp, ok := v.(*Response); ok {
			return resp, nil
		} else if err, ok := v.(error); ok {
			if firstError == nil {
				firstError = err
			}
		}

		if total >= totalUrls {
			break
		}
	}

	if firstError != nil {
		return nil, firstError
	}

	return nil, fmt.Errorf("doh: all %d providers failed to respond", totalUrls)
}

func (c *DoH) updateCache(d Domain, t Type, resp *Response, s ...ECS) {
	var ss string
	if len(s) > 0 && s[0] != "" {
		ss = strings.TrimSpace(string(s[0]))
	}
	hasher := sha1.New()
	hasher.Write([]byte(string(d) + string(t) + ss))
	cacheKey := hex.EncodeToString(hasher.Sum(nil))
	ttl := 30
	if len(resp.Answer) > 0 {
		ttl = resp.Answer[0].TTL
	}
	c.cache.Set(cacheKey, resp, int64(ttl))
}

// query builds and executes a DoH query.
func (c *DoH) query(ctx context.Context, u string, d Domain, t Type, s ...ECS) (*Response, error) {
	req, err := c.buildRequest(ctx, u, d, t, s...)
	if err != nil {
		return nil, err
	}

	return c.doRequest(req)
}

// buildRequest creates a new HTTP request for the given DoH query parameters.
func (c *DoH) buildRequest(ctx context.Context, u string, d Domain, t Type, s ...ECS) (*http.Request, error) {
	name, err := d.Punycode()
	if err != nil {
		return nil, fmt.Errorf("doh: failed to convert domain to punycode: %w", err)
	}

	param := url.Values{}
	param.Add("name", name)
	param.Add("type", strings.TrimSpace(string(t)))

	if len(s) > 0 {
		ss := strings.TrimSpace(string(s[0]))
		if ss != "" {
			ss, err := FixSubnet(ss)
			if err != nil {
				return nil, fmt.Errorf("doh: invalid client subnet: %w", err)
			}
			param.Add("edns_client_subnet", ss)
		}
	}

	dnsURL := fmt.Sprintf("%s?%s", u, param.Encode())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, dnsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("doh: failed to create http request for %s: %w", dnsURL, err)
	}

	req.Header.Set("Accept", "application/dns-json")
	req.Header.Set("User-Agent", fmt.Sprintf("%s/%s (%s)", Name, Version, Source))

	return req, nil
}

// doRequest executes the HTTP request and parses the response.
func (c *DoH) doRequest(req *http.Request) (*Response, error) {
	u := req.URL.String()
	rsp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("doh: http request failed for %s: %w", u, err)
	}
	defer func() {
		_ = rsp.Body.Close()
	}()

	if rsp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("doh: provider %s returned unexpected status code: %d", u, rsp.StatusCode)
	}

	data, err := io.ReadAll(rsp.Body)
	if err != nil {
		return nil, fmt.Errorf("doh: failed to read response body from %s: %w", u, err)
	}

	rr := &Response{
		Provider: u,
	}

	err = json.Unmarshal(data, rr)
	if err != nil {
		return nil, fmt.Errorf("doh: failed to unmarshal json response from %s: %w", u, err)
	}

	if rr.Status != 0 {
		return rr, fmt.Errorf("doh: provider %s returned error in dns response (status: %d)", u, rr.Status)
	}

	return rr, nil
}
