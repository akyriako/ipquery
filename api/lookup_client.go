package api

import (
	"net"
	"net/http"
	"strings"
)

type LookupClient struct {
	TrustedProxies []*net.IPNet
	AsnReader      *AsnReader
	CityReader     *CityReader
	RiskChecker    *AbuseIpDbChecker
}

func (c *LookupClient) GetClientIP(r *http.Request) string {
	remoteIP := c.remoteAddrIP(r.RemoteAddr)
	if remoteIP == nil {
		return ""
	}

	// Only trust forwarded headers if the direct peer is a trusted proxy.
	if c.isTrustedProxy(remoteIP) {
		// Cloudflare (optional)
		if ip := c.parseIP(r.Header.Get("CF-Connecting-IP")); ip != nil {
			return ip.String()
		}

		// Pangolin/Caddy gives the real client here in your setup
		if ip := c.parseIP(r.Header.Get("X-Real-IP")); ip != nil {
			return ip.String()
		}

		// RFC7239 (optional)
		if ip := c.parseForwardedFor(r.Header.Get("Forwarded")); ip != nil {
			return ip.String()
		}

		// XFF last, because yours currently contains tunnel IPs
		if ip := c.firstIPFromXFF(r.Header.Get("X-Forwarded-For")); ip != nil {
			return ip.String()
		}
	}

	return remoteIP.String()
}

func (c *LookupClient) isTrustedProxy(remoteIP net.IP) bool {
	for _, n := range c.TrustedProxies {
		if n.Contains(remoteIP) {
			return true
		}
	}
	return false
}

func (c *LookupClient) firstIPFromXFF(xff string) net.IP {
	if xff == "" {
		return nil
	}
	parts := strings.Split(xff, ",")
	for _, p := range parts {
		ip := c.parseIP(strings.TrimSpace(p))
		if ip != nil {
			return ip
		}
	}
	return nil
}

func (c *LookupClient) remoteAddrIP(remoteAddr string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddr)
	if err == nil {
		return c.parseIP(host)
	}
	// Sometimes already an IP
	return c.parseIP(remoteAddr)
}

func (c *LookupClient) parseIP(s string) net.IP {
	ip := net.ParseIP(strings.TrimSpace(s))
	if ip == nil {
		return nil
	}
	// Normalize IPv4-in-IPv6 form
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip
}

func (c *LookupClient) parseForwardedFor(v string) net.IP {
	// Forwarded: for=203.0.113.60;proto=https;by=...
	// Can be comma-separated; we take the first entry.
	if v == "" {
		return nil
	}
	first := strings.Split(v, ",")[0]

	low := strings.ToLower(first)
	i := strings.Index(low, "for=")
	if i < 0 {
		return nil
	}

	s := first[i+4:]
	if j := strings.IndexByte(s, ';'); j >= 0 {
		s = s[:j]
	}
	s = strings.TrimSpace(s)
	s = strings.Trim(s, `"`)
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")

	// Sometimes includes port
	if host, _, err := net.SplitHostPort(s); err == nil {
		s = host
	}

	ip := net.ParseIP(s)
	if ip == nil {
		return nil
	}
	if v4 := ip.To4(); v4 != nil {
		return v4
	}
	return ip
}
