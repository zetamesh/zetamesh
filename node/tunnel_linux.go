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
	"os/exec"

	"github.com/pkg/errors"
	"github.com/songgao/water"
)

func (n *Node) setupTunnel() error {
	// Initialize the virtual interface
	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		return errors.WithMessage(err, "interface initialize failed")
	}

	// Set the IP address for the virtual interface
	source := n.opt.Address
	ifconfig := exec.Command("ip", "addr", "add", fmt.Sprintf("%s/16", source), "dev", ifce.Name())
	if out, err := ifconfig.CombinedOutput(); err != nil {
		return errors.WithMessagef(err, "output: %s", string(out))
	}

	// Up the virtual interface device
	upifce := exec.Command("ip", "link", "set", "dev", ifce.Name(), "up")
	if out, err := upifce.CombinedOutput(); err != nil {
		return errors.WithMessagef(err, "output: %s", string(out))
	}

	n.veth = ifce
	return nil
}
