package controller

import (
	"context"
	"encoding/json"
	"net"

	"github.com/aryanjand/Unix-Password-Cracker/internal/protocol"
)

type Conn struct {
	conn   net.Conn
	stop   context.Context
	cancel context.CancelFunc
	send   chan protocol.Message
	recv   chan protocol.Message
}

func NewConn(conn net.Conn) *Conn {
	ctx, cancel := context.WithCancel(context.Background())

	cc := &Conn{
		conn:   conn,
		stop:   ctx,
		cancel: cancel,
		send:   make(chan protocol.Message, 32),
		recv:   make(chan protocol.Message, 32),
	}

	go cc.ReadLoop()
	go cc.WriteLoop()

	return cc

}

func (c *Conn) ReadLoop() error {
	decoder := json.NewDecoder(c.conn)

	for {
		var msg protocol.Message
		if err := decoder.Decode(&msg); err != nil {
			c.Close()
			return err
		}

		select {

		case <-c.stop.Done():
			return nil

		case c.recv <- msg:

		}

	}
}

func (c *Conn) WriteLoop() error {
	encoder := json.NewEncoder(c.conn)

	for {
		select {
		case <-c.stop.Done():
			return nil

		case msg := <-c.send:
			if err := encoder.Encode(msg); err != nil {
				c.Close()
				return err
			}

		}
	}
}

func (c *Conn) Close() {
	c.cancel()
	_ = c.conn.Close()
}
