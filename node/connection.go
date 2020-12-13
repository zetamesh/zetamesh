// Copyright 2020 ZetaMesh Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// See the License for the specific language governing permissions and
// limitations under the License.

package node

import (
	"fmt"
	"net"
	"time"

	"github.com/lonng/zetamesh/codec"
	"github.com/lonng/zetamesh/constant"
	"github.com/lonng/zetamesh/message"
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type handler interface {
	handlePacket(remote net.Addr, data []byte)
	handleClosed(conn *connection)
}

type connectionState byte

const (
	// StateConnecting represents the state of connecting to the peer
	StateConnecting connectionState = 1
	// StateEstablished represents the state of connection has been established
	StateEstablished connectionState = 2
)

// String implements the fmt.Stringer interface
func (s connectionState) String() string {
	switch s {
	case StateConnecting:
		return "StateConnecting"
	case StateEstablished:
		return "StateEstablished"
	default:
		return fmt.Sprintf("UnknowState(%d)", s)
	}
}

type connection struct {
	selfVirtAddr string
	peerVirtAddr string
	handler      handler
	once         atomic.Bool
	state        connectionState
	peer         net.Conn
	pipeline     chan []byte
	keepalive    time.Time
	die          chan struct{}
}

func (c *connection) loop() {
	if c.once.Swap(true) {
		return
	}

	var (
		// The duration will be reset according to the current state of this connection
		connecting = time.After(0)

		// Keepalive with the remote peer
		keepalive = time.NewTicker(constant.PeerKeepaliveDuration)

		send = func(data []byte) {
			if _, err := c.peer.Write(data); err != nil {
				zap.L().Error("Send message failed", zap.Error(err), zap.Int("state", int(c.state)))
			}
		}

		ping = func() {
			data := codec.Encode(message.PacketType_Ping, &message.CtrlPing{
				VirtAddress: c.selfVirtAddr,
				Nonce:       randseq(128),
			})
			send(data)
		}
	)

	defer keepalive.Stop()
	defer c.handler.handleClosed(c)

	go c.read()

	for {
		select {
		case <-connecting:
			if c.state != StateConnecting {
				continue
			}
			connecting = time.After(constant.ConnectingRetryDuration)
			ping()

		case <-keepalive.C:
			// Keepalive timeout and close currently connection to wait reconnect
			if c.keepalive.Add(constant.PeerKeepaliveDuration * 2 / 3).Before(time.Now()) {
				c.close()
				continue
			}
			if c.state != StateEstablished {
				continue
			}
			ping()

		case data := <-c.pipeline:
			send(data)

		case <-c.die:
			_ = c.peer.Close()
			zap.L().Info("Connection closed", zap.String("peer", c.peerVirtAddr), zap.Stringer("desination", c.peer.RemoteAddr()))
			return
		}
	}
}

func (c *connection) read() {
	buffer := make([]byte, 4096)
	for {
		n, err := c.peer.Read(buffer)
		if err != nil {
			zap.L().Info("Read peer connection failed", zap.Error(err))
			return
		}
		c.handler.handlePacket(c.peer.RemoteAddr(), buffer[:n])
	}
}

func (c *connection) close() {
	select {
	case <-c.die:
	default:
		close(c.die)
	}
}
