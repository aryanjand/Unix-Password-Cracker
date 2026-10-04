package tcp

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"sync"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

// ErrConnClosed is returned by SendMsg once the connection has stopped
// accepting outbound messages (Close has been called, or the connection
// died).
var ErrConnClosed = errors.New("connection closed")

type Conn struct {
	Conn   net.Conn
	Stop   context.Context
	Cancel context.CancelFunc
	Recv   chan protocol.Message

	writeMu   sync.Mutex
	closed    bool
	closeOnce sync.Once
}

func NewConn(conn net.Conn) *Conn {
	stopCtx, cancel := context.WithCancel(context.Background())

	cc := &Conn{
		Conn:   conn,
		Stop:   stopCtx,
		Cancel: cancel,
		Recv:   make(chan protocol.Message, 32),
	}

	go cc.ReadLoop()
	return cc
}

func (c *Conn) SendMsg(msg protocol.Message) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if c.closed {
		return ErrConnClosed
	}

	if err := json.NewEncoder(c.Conn).Encode(msg); err != nil {
		c.closed = true
		c.teardown()
		return err
	}
	return nil
}

func (c *Conn) ReadLoop() {
	decoder := json.NewDecoder(c.Conn)

	for {
		var msg protocol.Message
		if err := decoder.Decode(&msg); err != nil {
			c.Close()
			return
		}

		select {
		case <-c.Stop.Done():
			return
		case c.Recv <- msg:
		}
	}
}

func (c *Conn) Close() {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.closed = true
	c.teardown()
}

func (c *Conn) teardown() {
	c.closeOnce.Do(func() {
		c.Cancel()
		_ = c.Conn.Close()
	})
}
