package transport

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"kv-cache/internal/command"
	storage "kv-cache/internal/storage"
)

func TestServer_SET_GET(t *testing.T) {
	store := storage.NewMemoryStore()
	executor := command.NewExecutor(store, nil)
	server := NewServer(executor, ":0")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	addr := server.ln.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// SET test
	req := "*3\r\n$3\r\nSET\r\n$4\r\ntest\r\n$5\r\nvalue\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "+OK\r\n" {
		t.Fatalf("unexpected SET response: %q", string(buf[:n]))
	}

	// GET test
	req2 := "*2\r\n$3\r\nGET\r\n$4\r\ntest\r\n"
	if _, err := conn.Write([]byte(req2)); err != nil {
		t.Fatal(err)
	}

	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err = conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	expected := "$5\r\nvalue\r\n"
	if string(buf[:n]) != expected {
		t.Fatalf("unexpected GET response: %q, expected %q", string(buf[:n]), expected)
	}
}

func TestServer_QUIT(t *testing.T) {
	store := storage.NewMemoryStore()
	executor := command.NewExecutor(store, nil)
	server := NewServer(executor, ":0")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	addr := server.ln.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}

	req := "*1\r\n$4\r\nQUIT\r\n"
	if _, err := conn.Write([]byte(req)); err != nil {
		t.Fatal(err)
	}

	buf := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatal(err)
	}
	if string(buf[:n]) != "+OK\r\n" {
		t.Fatalf("unexpected QUIT response: %q", string(buf[:n]))
	}

	// Connection should be closed by server
	_, err = conn.Read(buf)
	if err == nil {
		t.Fatal("expected connection to be closed")
	}
}

func TestServer_Hash(t *testing.T) {
	store := storage.NewMemoryStore()
	executor := command.NewExecutor(store, nil)
	server := NewServer(executor, ":0")
	if err := server.Start(); err != nil {
		t.Fatal(err)
	}
	defer server.Close()

	addr := server.ln.Addr().String()
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	send := func(cmd string) string {
		if _, err := conn.Write([]byte(cmd)); err != nil {
			t.Fatal(err)
		}
		buf := make([]byte, 4096)
		conn.SetReadDeadline(time.Now().Add(10 * time.Second))
		n, err := conn.Read(buf)
		if err != nil {
			return fmt.Sprintf("read error: %v", err)
		}
		return string(buf[:n])
	}

	if got := send("*4\r\n$4\r\nHSET\r\n$2\r\nh1\r\n$2\r\nf1\r\n$4\r\nv111\r\n"); !strings.HasPrefix(got, ":1") {
		t.Fatalf("unexpected HSET response: %q", got)
	}
	if got := send("*3\r\n$4\r\nHGET\r\n$2\r\nh1\r\n$2\r\nf1\r\n"); got != "$4\r\nv111\r\n" {
		t.Fatalf("unexpected HGET response: %q", got)
	}
	if got := send("*2\r\n$7\r\nHGETALL\r\n$2\r\nh1\r\n"); !strings.HasPrefix(got, "*2") {
		t.Fatalf("unexpected HGETALL response: %q", got)
	}
	if got := send("*3\r\n$4\r\nHDEL\r\n$2\r\nh1\r\n$2\r\nf1\r\n"); !strings.HasPrefix(got, ":1") {
		t.Fatalf("unexpected HDEL response: %q", got)
	}
}
