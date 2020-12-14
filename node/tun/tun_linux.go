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

package tun

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"unsafe"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// NewTUN creates a new TUN device and set the address to the specified address
func NewTUN(addr string) (Device, error) {
	fd, err := syscall.Open("/dev/net/tun", os.O_RDWR|syscall.O_NONBLOCK, 0)
	if err != nil {
		return nil, err
	}

	const size = unix.IFNAMSIZ + 64

	var setiff [size]byte
	var flags uint16 = unix.IFF_TUN | unix.IFF_NO_PI
	*(*uint16)(unsafe.Pointer(&setiff[unix.IFNAMSIZ])) = flags

	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(unix.TUNSETIFF),
		uintptr(unsafe.Pointer(&setiff[0])),
	)
	if errno != 0 {
		return nil, errno
	}

	name := strings.Trim(string(setiff[:unix.IFNAMSIZ]), "\x00")

	// Set MTU
	setmtu := func() error {
		// open datagram socket
		fd, err := unix.Socket(
			unix.AF_INET,
			unix.SOCK_DGRAM,
			0,
		)

		if err != nil {
			return err
		}

		defer unix.Close(fd)

		var setmtu [size]byte
		copy(setmtu[:], name)
		*(*uint32)(unsafe.Pointer(&setmtu[unix.IFNAMSIZ])) = uint32(DefaultMTU)

		_, _, errno := unix.Syscall(
			unix.SYS_IOCTL,
			uintptr(fd),
			uintptr(unix.SIOCSIFMTU),
			uintptr(unsafe.Pointer(&setmtu[0])),
		)

		if errno != 0 {
			return errors.New("failed to set MTU of TUN device")
		}
		return nil
	}
	if err := setmtu(); err != nil {
		return nil, err
	}

	dev := &generalDevice{
		name:            name,
		ReadWriteCloser: os.NewFile(uintptr(fd), "tun"),
	}

	// Set the IP address for the virtual interface
	source := addr
	ifconfig := exec.Command("ip", "addr", "add", fmt.Sprintf("%s/16", source), "dev", name)
	if out, err := ifconfig.CombinedOutput(); err != nil {
		return nil, errors.WithMessagef(err, "output: %s", string(out))
	}

	// Up the virtual interface device
	upifce := exec.Command("ip", "link", "set", "dev", name, "up")
	if out, err := upifce.CombinedOutput(); err != nil {
		return nil, errors.WithMessagef(err, "output: %s", string(out))
	}

	return dev, nil
}
