// Entry point of the VPN program
// By CLAREMBEAU Alexis & FLORIOT Remi
// 2017-2018 Master's thesis. All rights reserved.

package main

import (
	"errors"
	"quic_utils"
	. "quic_vpn/internal"
)

var (
	errUnknownMode = errors.New("unknown program mode")
)

func main() {

	conf := VpnConfig{
		Mtu:           1150,
		Iface_type:    "tun",
		Iface_name:    "tuntap",
		Multi_streams: false,
	}
	err := conf.Parse()
	quic_utils.Check(err)

	switch conf.Mode {
	case "client":
		s, err := NewClientInstance(&conf)

		quic_utils.Check(err)
		quic_utils.Check(s.Run())

		break
	case "server":
		s, err := NewServerInstance(&conf)

		quic_utils.Check(err)
		quic_utils.Check(s.Run())
		break
	default:
		quic_utils.Check(errUnknownMode)
		break
	}
}
