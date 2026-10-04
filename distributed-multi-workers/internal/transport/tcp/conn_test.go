package tcp

import (
	"encoding/json"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

// TestConn_SendMsgCompletesBeforeClose is a regression test for shutdown:
// a message handed to SendMsg must hit the wire, and Close must wait for
// that Encode (via writeMu) rather than tearing the socket down under it.
func TestConn_SendMsgCompletesBeforeClose(t *testing.T) {
	local, remote := net.Pipe()
	t.Cleanup(func() { _ = remote.Close() })

	cc := NewConn(local)

	// net.Pipe is unbuffered, so Encode blocks until the peer reads.
	errCh := make(chan error, 1)
	go func() {
		errCh <- cc.SendMsg(protocol.Message{Command: protocol.MsgStopAck})
	}()

	decoder := json.NewDecoder(remote)
	var got protocol.Message
	if err := decoder.Decode(&got); err != nil {
		t.Fatalf("decode message: %v", err)
	}
	if got.Command != protocol.MsgStopAck {
		t.Fatalf("got command %q, want %q", got.Command, protocol.MsgStopAck)
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("SendMsg: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("SendMsg did not return after the message was read")
	}

	cc.Close()

	if err := cc.SendMsg(protocol.Message{Command: protocol.MsgStop}); err != ErrConnClosed {
		t.Fatalf("SendMsg after Close = %v, want ErrConnClosed", err)
	}
}

// TestConn_CloseIsIdempotent ensures concurrent/repeated Close calls (as
// happen in practice — e.g. a read error and an explicit Close racing)
// never panic or hang.
func TestConn_CloseIsIdempotent(t *testing.T) {
	local, remote := net.Pipe()
	t.Cleanup(func() { _ = remote.Close() })

	cc := NewConn(local)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			cc.Close()
		}()
	}

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("concurrent Close calls did not all return")
	}

	_ = remote.Close()
}
