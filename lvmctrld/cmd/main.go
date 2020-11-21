// Copyright 2020 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/aleofreddi/csi-sanlock-lvm/lvmctrld/pkg"
	"k8s.io/klog"
)

var (
	hostAddr = flag.String("lock-with-host-addr", "", "enable locking, compute host id from the given ip address. This options is mutually exclusive with lock-with-host-id")
	hostId   = flag.String("lock-with-host-id", "", "enable locking, use the given host id. This option is mutually exclusive with lock-with-host-addr")
	listen   = flag.String("listen", "tcp://0.0.0.0:9000", "listen address")
	version  string
	commit   string
)

func main() {
	klog.InitFlags(nil)
	flag.Parse()
	klog.Infof("Starting lvmctrld %s (%s)", version, commit)

	listener, err := bootstrap()
	if err != nil {
		klog.Errorf("Bootstrap failed: %v", err)
		os.Exit(2)
	}
	if err = listener.Run(); err != nil {
		klog.Errorf("Execution failed: %v", err)
		os.Exit(3)
	}
	os.Exit(0)
}

func bootstrap() (*lvmctrld.Listener, error) {
	// Parse host id
	var id uint16
	var err error
	if *hostId != "" || *hostAddr != "" {
		id, err = parseHostId(*hostId, *hostAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse host id: %v", err)
		}
		if err := lvmctrld.StartLock(id, []string{}); err != nil {
			return nil, fmt.Errorf("failed to start lock: %v", err)
		}
	}

	// Start server
	listener, err := lvmctrld.NewListener(*listen, id)
	if err != nil {
		return nil, fmt.Errorf("failed to instance listener: %v", err)
	}
	if err = listener.Init(); err != nil {
		return nil, fmt.Errorf("failed to initialize listener: %v", err)
	}
	return listener, nil
}

func parseHostId(hostId string, hostAddr string) (uint16, error) {
	if hostId != "" && hostAddr != "" {
		return 0, fmt.Errorf("host id and node ip are mutually exclusive")
	}
	if hostId == "" && hostAddr == "" {
		return 0, fmt.Errorf("host id or node ip required")
	}

	if hostId != "" {
		v, err := strconv.ParseInt(hostId, 10, 16)
		if err != nil || v < 1 || v > 2000 {
			return 0, fmt.Errorf("invalid host id %s, expected a decimal in range [1, 2000]", hostId)
		}
		return uint16(v), nil
	}
	v, err := addressToHostId(hostAddr)
	if err != nil {
		return 0, fmt.Errorf("invalid address")
	}
	return v, nil
}

func addressToHostId(ip string) (uint16, error) {
	address := net.ParseIP(ip)
	if address == nil {
		return 0, fmt.Errorf("invalid ip address: %s", ip)
	}
	return uint16((binary.BigEndian.Uint32(address[len(address)-4:])&0x7ff)%1999 + 1), nil
}
