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
	"go.uber.org/zap"
)

// TODO: https://git.zx2c4.com/wintun/
func (n *Node) setupTunnel() error {
	// Initialize the virtual interface
	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		return err
	}

	zap.L().Info("Virtual interface name", zap.String("name", ifce.Name()))

	// Set the IP address for the virtual interface
	source := n.opt.Address
	ifconfig := exec.Command("netsh", "interface", "ip", "set", "address",
		fmt.Sprintf(`name="%s"`, ifce.Name()),
		fmt.Sprintf("addr=%s", source),
		fmt.Sprintf("mask=%s", "255.255.0.0"), "gateway=none")
	if out, err := ifconfig.CombinedOutput(); err != nil {
		return errors.WithMessagef(err, "output: %s", string(out))
	}

	n.veth = ifce
	return nil
}
