package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"testing"
	"time"
)

func TestServeHTTPBindFailureReturnsError(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	port := ln.Addr().(*net.TCPAddr).Port
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv := &http.Server{
		Addr:    addr,
		Handler: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}),
	}
	cfg := serveConfig{Host: "127.0.0.1", Port: strconv.Itoa(port)}

	errCh := make(chan error, 1)
	go func() {
		errCh <- serveHTTP(ctx, srv, cfg, cancel)
	}()

	select {
	case err := <-errCh:
		if err == nil {
			t.Fatal("expected bind error, got nil")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serveHTTP did not return on bind failure")
	}
}

func TestServeHTTPServesUntilCancel(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()

	ctx, cancel := context.WithCancel(context.Background())
	addr := net.JoinHostPort("127.0.0.1", strconv.Itoa(port))
	srv := &http.Server{
		Addr: addr,
		Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_, _ = fmt.Fprint(w, "ok")
		}),
	}
	cfg := serveConfig{Host: "127.0.0.1", Port: strconv.Itoa(port)}

	errCh := make(chan error, 1)
	go func() {
		errCh <- serveHTTP(ctx, srv, cfg, cancel)
	}()

	deadline := time.Now().Add(3 * time.Second)
	var lastErr error
	for {
		if time.Now().After(deadline) {
			cancel()
			t.Fatalf("never reached server: %v", lastErr)
		}
		resp, err := http.Get("http://" + addr + "/")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(20 * time.Millisecond)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("serveHTTP after cancel: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("serveHTTP did not return after cancel")
	}
}
