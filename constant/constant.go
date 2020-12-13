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

package constant

import "time"

// HeartbeatInterval represent the interval of keepping heartbeat with
// the gateway in seconds
const HeartbeatInterval = 20

// MaxBufferSize represents the max buffer size of read UDP packet
const MaxBufferSize = 4096

// MaxRetrySend represents the max tries of send notification to the peer
const MaxRetrySend = 10

// ConnectingRetryDuration represents the interval of retrying send
// initial Ping packet to the peer
const ConnectingRetryDuration = 100 * time.Millisecond

// PeerKeepaliveDuration represents the interval of keepaliving heartbeat
const PeerKeepaliveDuration = 5 * time.Second
