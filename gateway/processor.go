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

package gateway

import (
	"net"

	"github.com/lonng/zetamesh/api"
	"github.com/lonng/zetamesh/codec"
	"github.com/lonng/zetamesh/message"
	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

var protos = [...]proto.Message{
	message.PacketType_Heartbeat:     &message.CtrlHeartbeat{},
	message.PacketType_OpenTunnelAck: &message.CtrlOpenTunnelAck{},
	message.PacketType_Relay:         &message.CtrlRelay{},
}

type processor struct {
	server   *api.Server
	notifier *notifier
}

func newProcessor(server *api.Server, notifier *notifier) *processor {
	return &processor{
		server:   server,
		notifier: notifier,
	}
}

func (p *processor) process(addr *net.UDPAddr, data []byte) error {
	packetType := message.PacketType(data[0])
	if int(packetType) >= len(protos) {
		return errors.Errorf("unrecognized message type: %d", packetType)
	}

	protoType := protos[int(packetType)]
	if protoType == nil {
		return nil
	}
	protoType.(codec.Reseter).Reset()
	err := proto.Unmarshal(data[1:], protoType)
	if err != nil {
		return errors.WithMessagef(err, "unmarshal message %s failed", packetType)
	}

	switch packetType {
	case message.PacketType_Heartbeat:
		heartbeat := protoType.(*message.CtrlHeartbeat)
		if heartbeat.VirtAddress == "" {
			return nil
		}
		p.server.Heartbeat(addr, heartbeat)

	case message.PacketType_OpenTunnelAck:
		ack := protoType.(*message.CtrlOpenTunnelAck)
		p.notifier.openTunnelAck(ack.AckId)

	case message.PacketType_Relay:
		relay := protoType.(*message.CtrlRelay)
		dst := p.server.Peer(relay.VirtAddress)
		if dst == nil {
			return errors.Errorf("destination peer '%s' not found", relay.VirtAddress)
		}
		p.notifier.relay(dst.UDPAddress, relay.Data)
	}

	return nil
}
