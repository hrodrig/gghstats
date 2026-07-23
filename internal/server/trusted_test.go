package server

import (
	"net"
	"testing"
)

func TestParseTrustedProxies(t *testing.T) {
	t.Run("empty", func(t *testing.T) {
		tp := ParseTrustedProxies("  ")
		if tp == nil || tp.ContainsIP(net.ParseIP("10.0.0.1")) {
			t.Fatalf("empty must not contain any IP; tp=%v", tp)
		}
	})
	t.Run("single ip becomes /32", func(t *testing.T) {
		tp := ParseTrustedProxies("127.0.0.1")
		if !tp.ContainsIP(net.ParseIP("127.0.0.1")) {
			t.Fatal("expected 127.0.0.1 contained")
		}
		if tp.ContainsIP(net.ParseIP("127.0.0.2")) {
			t.Fatal("did not expect 127.0.0.2")
		}
	})
	t.Run("ipv6 single becomes /128", func(t *testing.T) {
		tp := ParseTrustedProxies("::1")
		if !tp.ContainsIP(net.ParseIP("::1")) {
			t.Fatal("expected ::1 contained")
		}
	})
	t.Run("skips invalid", func(t *testing.T) {
		tp := ParseTrustedProxies("not-an-ip, 10.0.0.0/8")
		if !tp.ContainsIP(net.ParseIP("10.1.2.3")) {
			t.Fatal("expected 10.1.2.3 in 10.0.0.0/8")
		}
	})
}

func TestTrustedProxiesContains(t *testing.T) {
	tp := ParseTrustedProxies("10.0.0.0/8, 192.168.0.0/16")
	if tp.ContainsIP(nil) {
		t.Fatal("nil IP must not match")
	}
	var nilTP *TrustedProxies
	if nilTP.ContainsIP(net.ParseIP("10.0.0.1")) {
		t.Fatal("nil receiver must not contain IP")
	}
	if !tp.ContainsIP(net.ParseIP("10.255.255.255")) {
		t.Fatal("expected 10.255.255.255 in 10.0.0.0/8")
	}
	if !tp.ContainsIP(net.ParseIP("192.168.1.1")) {
		t.Fatal("expected 192.168.1.1 in 192.168.0.0/16")
	}
	if tp.ContainsIP(net.ParseIP("8.8.8.8")) {
		t.Fatal("did not expect 8.8.8.8")
	}
}
