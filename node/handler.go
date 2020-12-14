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
	"context"
	"net"
	"time"

	"github.com/lonng/zetamesh/codec"
	"github.com/lonng/zetamesh/constant"
	"github.com/lonng/zetamesh/message"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

var protos = []proto.Message{
	message.PacketType_Ping:       &message.CtrlPing{},
	message.PacketType_Pong:       &message.CtrlPong{},
	message.PacketType_OpenTunnel: &message.CtrlOpenTunnel{},
}

func (n *Node) schedule(ctx context.Context) error {
	buffer := make([]byte, constant.MaxBufferSize)
	for {
		select {
		case <-ctx.Done():
			zap.L().Info("Context cancelled", zap.Error(ctx.Err()))
			return ctx.Err()

		default:
			// Read new UDP message
			c, remote, err := n.gateway.ReadFromUDP(buffer)
			if err != nil {
				zap.L().Error("Read UDP failed", zap.Error(err))
				continue
			}

			n.handlePacket(remote, buffer[:c])
		}
	}
}

func (n *Node) handlePacket(remote net.Addr, data []byte) {
	// Invalid packet
	if len(data) < 1 {
		return
	}

	packetType := message.PacketType(data[0])
	payload := data[1:]
	if packetType == message.PacketType_Data {
		zap.L().Debug("Receive packet", zap.Stringer("source", remote))
		dataCopy := make([]byte, len(payload))
		copy(dataCopy, payload)
		n.pipeline <- dataCopy
		return
	}

	if int(packetType) >= len(protos) {
		zap.L().Error("Unrecognized message type", zap.Stringer("type", packetType), zap.Stringer("source", remote))
		return
	}

	msg := protos[int(packetType)]
	if reseter, ok := msg.(codec.Reseter); ok {
		reseter.Reset()
		err := proto.Unmarshal(payload, msg)
		if err != nil {
			zap.L().Error("Unmarshal proto message failed", zap.Stringer("type", packetType), zap.Error(err))
			return
		}
	}

	switch packetType {
	case message.PacketType_Ping:
		n.onPing(remote, msg.(*message.CtrlPing))

	case message.PacketType_Pong:
		n.onPong(remote, msg.(*message.CtrlPong))

	case message.PacketType_OpenTunnel:
		n.onOpenTunnel(msg.(*message.CtrlOpenTunnel))
	}
}

func (n *Node) onPing(source net.Addr, ping *message.CtrlPing) {
	conn, found := n.connections.Load(ping.VirtAddress)
	if !found {
		return
	}

	zap.L().Debug("Receive Ping message", zap.String("peer", ping.VirtAddress), zap.Stringer("source", source))

	data := codec.Encode(message.PacketType_Pong, &message.CtrlPong{
		VirtAddress: n.opt.Address,
		Nonce:       randseq(128),
	})
	conn.(*connection).pipeline <- data
}

func (n *Node) onPong(source net.Addr, pong *message.CtrlPong) {
	c, found := n.connections.Load(pong.VirtAddress)
	if !found {
		zap.L().Info("Receive unknown connection Pong message", zap.String("vaddr", pong.VirtAddress))
		return
	}

	zap.L().Debug("Receive Pong message", zap.String("peer", pong.VirtAddress), zap.Stringer("source", source))

	conn := c.(*connection)
	conn.keepalive = time.Now()
	if conn.state != StateEstablished {
		conn.state = StateEstablished
	}
}

func (n *Node) onOpenTunnel(openTunnel *message.CtrlOpenTunnel) {
	// Send new connection ACK to the gateway server
	openTunnelAck := func() {
		ack := codec.Encode(message.PacketType_OpenTunnelAck, &message.CtrlOpenTunnelAck{
			AckId: openTunnel.AckId,
		})

		_, err := n.gateway.Write(ack)
		if err != nil {
			zap.L().Error("Acknowledge open tunnel failed", zap.Error(err))
		}
	}

	// Close the previous connection if the remote UDP address changed
	if conn, found := n.connections.Load(openTunnel.VirtAddress); found {
		conn := conn.(*connection)
		if conn.peer.RemoteAddr().String() == openTunnel.UdpAddress {
			openTunnelAck()
			return
		}

		// Reconnect to the new UDP address if the UDP address of peer has changed
		conn.close()
	}

	peer, err := n.dialer.Dial("udp", openTunnel.UdpAddress)
	if err != nil {
		return
	}

	conn := &connection{
		selfVirtAddr: n.opt.Address,
		peerVirtAddr: openTunnel.VirtAddress,
		handler:      n,
		peer:         peer,
		state:        StateConnecting,
		pipeline:     make(chan []byte, 128),
		keepalive:    time.Now(),
		die:          make(chan struct{}),
	}
	n.connections.Store(openTunnel.VirtAddress, conn)
	go conn.loop()

	openTunnelAck()
}

func (n *Node) handleClosed(conn *connection) {
	n.connections.Delete(conn.peerVirtAddr)
}
