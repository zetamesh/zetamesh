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
	"sync"
	"time"

	"github.com/lonng/zetamesh/api"
	"github.com/lonng/zetamesh/codec"
	"github.com/lonng/zetamesh/constant"
	"github.com/lonng/zetamesh/message"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type packet struct {
	destination string // IP:PORT
	typ         message.PacketType
	message     interface{}
}

type retryPacket struct {
	packet
	counter int64
}

type notifier struct {
	queue chan packet
	ackID atomic.Int64

	mu   sync.Mutex
	data map[int64]retryPacket
	read chan struct{}
}

func newNotifier() *notifier {
	return &notifier{
		queue: make(chan packet, 16),
		data:  map[int64]retryPacket{},
		read:  make(chan struct{}, 16),
	}
}

func (n *notifier) notify() {
	select {
	case n.read <- struct{}{}:
	default:
	}
}

func (n *notifier) start(conn *net.UDPConn, concurrency int) {
	worker := func(ch chan packet) {
		for p := range ch {
			dest, err := net.ResolveUDPAddr("udp", p.destination)
			if err != nil {
				zap.L().Error("Unexpected destination address", zap.String("destination", p.destination))
				continue
			}

			var data []byte
			switch payload := p.message.(type) {
			case proto.Message:
				data = codec.Encode(p.typ, payload)
			case []byte:
				data = codec.EncodeRaw(payload)
			}
			_, err = conn.WriteToUDP(data, dest)
			if err != nil {
				zap.L().Error("Send message failed", zap.String("destination", p.destination), zap.Stringer("type", p.typ), zap.Error(err))
				continue
			}
		}
	}

	roundTrip := 0
	chs := make([]chan packet, concurrency)
	for i := 0; i < concurrency; i++ {
		chs[i] = make(chan packet, 256)
		go worker(chs[i])
	}

	// Dispatch a packet into send queue
	send := func(p packet) {
		chs[roundTrip%len(chs)] <- p
		roundTrip++
	}

	// Retry will put the retryData into the queue
	retry := func() {
		n.mu.Lock()
		defer n.mu.Unlock()
		for ackID, rp := range n.data {
			if rp.counter > constant.MaxRetrySend {
				delete(n.data, ackID)
				continue
			}
			send(rp.packet)
		}
	}

	retryTicker := time.NewTicker(300 * time.Millisecond)
	defer retryTicker.Stop()
	for {
		select {
		case <-retryTicker.C:
			retry()
		case <-n.read:
			retry()
		case p := <-n.queue:
			send(p)
		}
	}
}

func (n *notifier) OpenTunnel(src, dst *api.PeerInfo) {
	n.mu.Lock()
	peers := []*api.PeerInfo{src, dst}
	for i, p := range peers {
		ackID := n.ackID.Add(1)
		n.data[ackID] = retryPacket{
			packet: packet{
				destination: p.UDPAddress,
				typ:         message.PacketType_OpenTunnel,
				message: &message.CtrlOpenTunnel{
					AckId:       ackID,
					VirtAddress: peers[1-i].VirtAddress,
					UdpAddress:  peers[1-i].UDPAddress,
				},
			},
		}
	}
	defer n.mu.Unlock()

	n.notify()
}

func (n *notifier) openTunnelAck(ackID int64) {
	n.mu.Lock()
	delete(n.data, ackID)
	n.mu.Unlock()
}

func (n *notifier) relay(dest string, data []byte) {
	n.queue <- packet{
		destination: dest,
		typ:         message.PacketType_Data,
		message:     data,
	}
}
