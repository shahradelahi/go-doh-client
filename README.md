# Go DoH Client

[![MIT licensed](https://img.shields.io/badge/license-MIT-blue)](./LICENSE)
[![Build Status](https://github.com/shahradelahi/go-doh-client/actions/workflows/ci.yml/badge.svg)](https://github.com/shahradelahi/go-doh-client/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/shahradelahi/go-doh-client)](https://goreportcard.com/report/github.com/shahradelahi/go-doh-client)
[![GoDoc](https://pkg.go.dev/badge/github.com/shahradelahi/go-doh-client)](https://pkg.go.dev/github.com/shahradelahi/go-doh-client)

A Go client library for DNS over HTTPS (DoH), as specified in [RFC 8484](https://tools.ietf.org/html/rfc8484).

## Overview

Traditional DNS queries are unencrypted, making them easy for internet providers and other third parties to intercept, monitor, and modify. This vulnerability is often used for censorship, where queries to certain domains are blocked, or for DNS hijacking, where responses are altered to redirect users to different IP addresses.

DNS over HTTPS (DoH) enhances privacy and security by encrypting DNS traffic and sending it over HTTPS. This makes it significantly more difficult to block or manipulate, providing a reliable way to avoid censorship and ensure the integrity of DNS responses.

This library provides a simple, flexible, and efficient Go client to perform DoH queries against trusted public resolvers.

## Features

- **Simple API**: A straightforward interface for making DNS queries.
- **Provider Flexibility**: Use a default list of trusted DoH providers or supply your own via an option.
- **Automatic Failover**: Automatically selects the fastest, most reliable provider from your list.
- **Query Caching**: Built-in in-memory cache to reduce latency for repeated queries.
- **Customizable HTTP Client**: Use functional options to configure providers, timeouts, transports, and more.
- **Standard Library Dependencies**: No external dependencies outside of the Go standard library.

## Installation

```bash
go get -u github.com/shahradelahi/go-doh-client
```

## Usage

Here’s a quick example of how to perform a DNS lookup.

### Basic Query

This example initializes a client with a default list of public resolvers and performs an `A` record lookup. The client will automatically determine the fastest provider to use.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shahradelahi/go-doh-client"
)

func main() {
	// Initialize a client with default DoH providers.
	client := doh.New()
	defer client.Close()

	// Create a context with a timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Perform an A record lookup for example.com.
	resp, err := client.Query(ctx, "example.com", doh.TypeA)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	// Print the answers.
	for _, answer := range resp.Answer {
		fmt.Printf("%s -> %s\n", answer.Name, answer.Data)
	}
}
```

### Advanced Configuration

You can easily customize the client's behavior by passing in functional options, such as providing a specific list of DoH servers or setting a custom timeout.

```go
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/shahradelahi/go-doh-client"
)

func main() {
	// A custom list of DoH providers.
	customProviders := []string{
		"https://cloudflare-dns.com/dns-query",
		"https://dns.google/resolve",
	}

	// Initialize a client with custom providers and a 3-second timeout.
	client := doh.New(
		doh.WithProviders(customProviders),
		doh.WithTimeout(3*time.Second),
	)
	defer client.Close()

	// Create a context.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Perform an MX record lookup.
	resp, err := client.Query(ctx, "gmail.com", doh.TypeMX)
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	// Print the answers.
	for _, answer := range resp.Answer {
		fmt.Printf("%s -> %s\n", answer.Name, answer.Data)
	}
}
```

## License

[MIT](/LICENSE) © [Shahrad Elahi](https://github.com/shahradelahi)
