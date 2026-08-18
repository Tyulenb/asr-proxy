package infrawebsocket

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func newTestSession(t *testing.T) (*session, *websocket.Conn, func()) {
	t.Helper()

	serverConn := make(chan *websocket.Conn, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := (&websocket.Upgrader{}).Upgrade(w, r, nil)
		if err != nil {
			return
		}
		serverConn <- conn
	}))

	url := "ws" + strings.TrimPrefix(server.URL, "http")
	client, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		server.Close()
		t.Fatalf("Dial() error = %v", err)
	}

	var conn *websocket.Conn
	select {
	case conn = <-serverConn:
	case <-time.After(time.Second):
		_ = client.Close()
		server.Close()
		t.Fatal("server did not upgrade WebSocket connection")
	}

	return &session{conn: conn}, client, func() {
		_ = conn.Close()
		_ = client.Close()
		server.Close()
	}
}

func TestRunClosesInboundWhenOutboundIsClosed(t *testing.T) {
	sess, client, cleanup := newTestSession(t)
	defer cleanup()

	inbound := make(chan []byte)
	outbound := make(chan []byte)
	close(outbound)

	done := make(chan error, 1)
	go func() {
		done <- sess.Run(context.Background(), outbound, inbound)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Run() error = %v, want nil", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Run() did not finish after outbound channel closed")
	}

	if _, ok := <-inbound; ok {
		t.Fatal("inbound channel is open after session completion")
	}

	_ = client.Close()
}

func TestReadStartReturnsWhenContextIsCanceled(t *testing.T) {
	sess, _, cleanup := newTestSession(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := sess.ReadStart(ctx)
	if err == nil {
		t.Fatal("ReadStart() error = nil, want cancellation or read error")
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("ReadStart() took %v after context cancellation", elapsed)
	}
}
