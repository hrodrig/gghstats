package server

import (
	"log/slog"
	"net"
	"strings"
)

// TrustedProxies is the set of TCP peers allowed to supply X-Forwarded-For / X-Real-IP.
type TrustedProxies struct {
	nets []*net.IPNet
}

// ParseTrustedProxies parses comma-separated IPs/CIDRs.
// Bare IPv4 → /32; bare IPv6 → /128. Invalid entries are skipped.
// Empty input returns a non-nil empty set (ContainsIP always false).
func ParseTrustedProxies(s string) *TrustedProxies {
	tp := &TrustedProxies{}
	for _, raw := range strings.Split(s, ",") {
		cidr := strings.TrimSpace(raw)
		if cidr == "" {
			continue
		}
		if !strings.Contains(cidr, "/") {
			if strings.Contains(cidr, ":") {
				cidr += "/128"
			} else {
				cidr += "/32"
			}
		}
		_, n, err := net.ParseCIDR(cidr)
		if err != nil {
			continue
		}
		tp.nets = append(tp.nets, n)
	}
	return tp
}

// ContainsIP reports whether ip is inside any trusted network.
func (t *TrustedProxies) ContainsIP(ip net.IP) bool {
	if t == nil || ip == nil {
		return false
	}
	for _, n := range t.nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

func (t *TrustedProxies) empty() bool {
	return t == nil || len(t.nets) == 0
}

// ShouldWarnTrustedProxies reports whether startup should warn that client IP
// headers are ignored because no trusted proxies were configured.
func ShouldWarnTrustedProxies(trusted *TrustedProxies, rateLimitEnabled, whitelistActive bool) bool {
	if !rateLimitEnabled && !whitelistActive {
		return false
	}
	return trusted.empty()
}

// WarnTrustedProxiesIfNeeded emits a startup warning when middleware depends on
// client IP headers but GGHSTATS_TRUSTED_PROXIES is empty.
func WarnTrustedProxiesIfNeeded(trusted *TrustedProxies, rateLimitEnabled, whitelistActive bool) {
	if !ShouldWarnTrustedProxies(trusted, rateLimitEnabled, whitelistActive) {
		return
	}
	slog.Warn("Client IP headers (X-Forwarded-For / X-Real-IP) are ignored because GGHSTATS_TRUSTED_PROXIES is empty. Behind a reverse proxy, set GGHSTATS_TRUSTED_PROXIES to the proxy IP or CIDR so rate-limit and whitelist see the real client")
}
