package worker

import (
	"context"
	"encoding/json"
	"net"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

type Conn struct {
	Conn   net.Conn
	Stop   context.Context
	Cancel context.CancelFunc
	Send   chan protocol.Message
	Recv   chan protocol.Message
}

func NewConn(conn net.Conn) *Conn {
	stopCtx, cancel := context.WithCancel(context.Background())

	cc := &Conn{
		Conn:   conn,
		Stop:   stopCtx,
		Cancel: cancel,
		Send:   make(chan protocol.Message, 32),
		Recv:   make(chan protocol.Message, 32),
	}

	go cc.ReadLoop()
	go cc.WriteLoop()

	return cc
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

func (c *Conn) WriteLoop() error {
	encoder := json.NewEncoder(c.Conn)

	for {
		select {
		case <-c.Stop.Done():
			return nil

		case msg := <-c.Send:
			if err := encoder.Encode(msg); err != nil {
				c.Close()
				return err
			}
		}
	}
}

func (c *Conn) Close() {
	c.Cancel()
	_ = c.Conn.Close()
}
