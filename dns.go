package doh

import (
	"strings"

	"golang.org/x/net/idna"
)

// Domain is dns query domain
type Domain string

// Type is dns query type
type Type string

// ECS is the edns0-client-subnet option, for example: 1.2.3.4/24
type ECS string

// Question is dns query question
type Question struct {
	Name string `json:"name"`
	Type int    `json:"type"`
}

// Answer is dns query answer
type Answer struct {
	Name string `json:"name"`
	Type int    `json:"type"`
	TTL  int    `json:"TTL"`
	Data string `json:"data"`
}

// Response is dns query response
type Response struct {
	Status   int        `json:"Status"`
	TC       bool       `json:"TC"`
	RD       bool       `json:"RD"`
	RA       bool       `json:"RA"`
	AD       bool       `json:"AD"`
	CD       bool       `json:"CD"`
	Question []Question `json:"Question"`
	Answer   []Answer   `json:"Answer"`
	Provider string     `json:"provider"`
}

// Supported dns query type
var (
	TypeA     = Type("A")
	TypeAAAA  = Type("AAAA")
	TypeCNAME = Type("CNAME")
	TypeMX    = Type("MX")
	TypeTXT   = Type("TXT")
	TypeSPF   = Type("SPF")
	TypeNS    = Type("NS")
	TypeSOA   = Type("SOA")
	TypePTR   = Type("PTR")
	TypeANY   = Type("ANY")
)

// Punycode returns punycode of domain
func (d Domain) Punycode() (string, error) {
	name := strings.TrimSpace(string(d))

	return idna.New(
		idna.MapForLookup(),
		idna.Transitional(true),
		idna.StrictDomainName(false),
	).ToASCII(name)
}
