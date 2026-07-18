package alert

import (
	"bufio"
	"context"
	"net"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestParseSinksJSON_SMTP(t *testing.T) {
	getenv := func(k string) string {
		m := map[string]string{
			"GGHSTATS_SMTP_HOST":     "smtp.example.com",
			"GGHSTATS_SMTP_PORT":     "587",
			"GGHSTATS_SMTP_USER":     "alerts@example.com",
			"GGHSTATS_SMTP_PASSWORD": "secret",
			"GGHSTATS_SMTP_TO":       "a@x.com; b@y.com",
		}
		return m[k]
	}
	sinks, err := ParseSinksJSON(`[
	  {"type":"smtp","host_env":"GGHSTATS_SMTP_HOST","port_env":"GGHSTATS_SMTP_PORT",
	   "user_env":"GGHSTATS_SMTP_USER","password_env":"GGHSTATS_SMTP_PASSWORD",
	   "to_env":"GGHSTATS_SMTP_TO"}
	]`, getenv)
	if err != nil {
		t.Fatal(err)
	}
	if len(sinks) != 1 || sinks[0].Type != TypeSMTP {
		t.Fatalf("got %+v", sinks)
	}
	s := sinks[0]
	if s.SMTPHost != "smtp.example.com" || s.SMTPPort != 587 || s.SMTPFrom != "alerts@example.com" {
		t.Fatalf("smtp fields=%+v", s)
	}
	if len(s.SMTPTo) != 2 || s.SMTPTo[0] != "a@x.com" || s.SMTPTo[1] != "b@y.com" {
		t.Fatalf("to=%v", s.SMTPTo)
	}
}

func TestBuildSMTPMessage(t *testing.T) {
	msg := string(buildSMTPMessage("from@ex.com", []string{"a@x.com"}, "subj line", "hello\nworld"))
	for _, want := range []string{
		"From: from@ex.com",
		"To: a@x.com",
		"Subject: subj line",
		"hello",
		"world",
	} {
		if !strings.Contains(msg, want) {
			t.Fatalf("missing %q in:\n%s", want, msg)
		}
	}
}

func TestSMTP_Send_plain(t *testing.T) {
	srv := startFakeSMTP(t)
	defer srv.Close()

	s := &SMTP{
		Host: "127.0.0.1",
		Port: srv.port,
		From: "gghstats@example.com",
		To:   []string{"ops@example.com"},
	}
	p := Payload{
		Kind: KindTraffic, Version: "0.10.1", Repo: "hrodrig/pgwd",
		Metric: "stars", Window: "milestone", Value: "100", Rule: "crossed 100 (next 500)",
		When: time.Date(2026, 7, 18, 12, 0, 0, 0, time.UTC),
	}
	if err := s.Send(context.Background(), p); err != nil {
		t.Fatal(err)
	}
	msgs := srv.messages()
	if len(msgs) != 1 || !strings.Contains(string(msgs[0]), "crossed 100") {
		t.Fatalf("messages=%v", msgs)
	}
	if !strings.Contains(string(msgs[0]), "Subject: gghstats/") {
		t.Fatalf("subject missing in %s", msgs[0])
	}
}

func TestBuildSenders_SMTP(t *testing.T) {
	senders := BuildSenders([]ResolvedSink{{
		Type: TypeSMTP, SMTPHost: "h", SMTPPort: 25, SMTPFrom: "a@b.com", SMTPTo: []string{"c@d.com"},
	}}, nil)
	if len(senders) != 1 || senders[0].Type() != TypeSMTP {
		t.Fatalf("got %#v", senders)
	}
}

func TestFilterSinks_SMTP(t *testing.T) {
	got, err := FilterSinks([]ResolvedSink{
		{Type: TypeSlack, URL: "http://s"},
		{Type: TypeSMTP, SMTPHost: "h"},
	}, "smtp")
	if err != nil || len(got) != 1 || got[0].Type != TypeSMTP {
		t.Fatalf("got %+v err=%v", got, err)
	}
}

type fakeSMTPServer struct {
	t      *testing.T
	ln     net.Listener
	port   int
	mu     sync.Mutex
	msgs   [][]byte
	closed bool
}

func startFakeSMTP(t *testing.T) *fakeSMTPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	_, portStr, err := net.SplitHostPort(ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	port, err := strconv.Atoi(portStr)
	if err != nil {
		t.Fatal(err)
	}
	s := &fakeSMTPServer{t: t, ln: ln, port: port}
	go s.acceptLoop()
	t.Cleanup(func() { s.Close() })
	return s
}

func (s *fakeSMTPServer) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	s.closed = true
	_ = s.ln.Close()
}

func (s *fakeSMTPServer) messages() [][]byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([][]byte, len(s.msgs))
	copy(out, s.msgs)
	return out
}

func (s *fakeSMTPServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			s.mu.Lock()
			closed := s.closed
			s.mu.Unlock()
			if closed {
				return
			}
			return
		}
		go s.handleConn(conn)
	}
}

func (s *fakeSMTPServer) handleConn(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	write := func(line string) error {
		_, err := conn.Write([]byte(line + "\r\n"))
		return err
	}
	_ = write("220 fake.local ESMTP")
	var inData bool
	var dataBuf strings.Builder
	for {
		line, err := br.ReadString('\n')
		if err != nil {
			return
		}
		line = strings.TrimRight(line, "\r\n")
		if inData {
			if line == "." {
				s.mu.Lock()
				s.msgs = append(s.msgs, []byte(dataBuf.String()))
				s.mu.Unlock()
				inData = false
				dataBuf.Reset()
				_ = write("250 OK")
				continue
			}
			dataBuf.WriteString(line)
			dataBuf.WriteString("\r\n")
			continue
		}
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			_ = write("250-fake.local")
			_ = write("250 OK")
		case strings.HasPrefix(upper, "MAIL FROM"), strings.HasPrefix(upper, "RCPT TO"):
			_ = write("250 OK")
		case upper == "DATA":
			_ = write("354 End data with <CR><LF>.<CR><LF>")
			inData = true
		case upper == "QUIT":
			_ = write("221 Bye")
			return
		case upper == "RSET":
			_ = write("250 OK")
		default:
			_ = write("502 not implemented")
		}
	}
}
