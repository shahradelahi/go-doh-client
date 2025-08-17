package doh

import (
	"fmt"
	"net/netip"
	"strings"
)

// FixSubnet ensures an IP string has a subnet mask, applying a default
// if one is not present. It is suitable for EDNS Client Subnet (ECS).
func FixSubnet(s string) (string, error) {
	s = strings.TrimSpace(s)

	// First, try to parse as a full, valid prefix.
	if p, err := netip.ParsePrefix(s); err == nil {
		// This handles valid CIDR notations like "1.2.3.4/24" or "8.8.8.8/32".
		return p.String(), nil
	}

	// If that fails, it might be a bare IP ("1.2.3.4") or an IP with an
	// empty or invalid mask part (e.g., "1.2.3.4/", "1.2.3.4/abc").
	// We treat all of these cases as a bare IP and apply the default ECS mask.
	ipPart, _, _ := strings.Cut(s, "/")
	addr, err := netip.ParseAddr(ipPart)
	if err != nil {
		return "", fmt.Errorf("invalid IP address: %w", err) // The IP part itself is invalid
	}

	// Apply the default ECS mask for privacy.
	var bits int
	if addr.Is4() {
		bits = 24
	} else {
		bits = 56
	}

	p := netip.PrefixFrom(addr, bits)
	return p.String(), nil
}
