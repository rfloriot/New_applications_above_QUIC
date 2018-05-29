// Network flow handling
// By CLAREMBEAU Alexis & FLORIOT Remi
// 2017-2018 Master's thesis. All rights reserved.

package internal

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

// Find flow corresponding to an IP packet
func FindFlow(packet []byte) (gopacket.Flow, error) {
	var ipType gopacket.LayerType

	headerV6, err := ipv6.ParseHeader(packet)
	if err != nil {
		return gopacket.Flow{}, err
	}
	if headerV6.Version == 6 {
		ipType = layers.LayerTypeIPv6
	}

	headerV4, err := ipv4.ParseHeader(packet)
	if err != nil {
		return gopacket.Flow{}, err
	}
	if headerV4.Version == 4 {
		ipType = layers.LayerTypeIPv4
	}

	gopacketPacket := gopacket.NewPacket(packet, ipType, gopacket.Lazy)

	if gopacketPacket.TransportLayer() != nil {
		return gopacketPacket.TransportLayer().TransportFlow(), nil
	} else {

		return gopacket.Flow{}, nil
	}
}

// temporal structure = gopacket.Flow with public fields
type encodableFlow struct {
	FlowType gopacket.EndpointType
	FlowSrc  []byte
	FlowDst  []byte
}

// Encode a flow to a string (to transfer over network)
func EncodeFlow(msource gopacket.Flow) string {
	m := encodableFlow{
		FlowType: msource.EndpointType(),
		FlowSrc:  msource.Src().Raw(),
		FlowDst:  msource.Dst().Raw(),
	}

	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(m)
	if err != nil {
		fmt.Println(`failed gob Encode`, err)
	}
	return base64.StdEncoding.EncodeToString(b.Bytes())
}

// Decode a flow from a string (transfered from the network)
func DecodeFlow(str string) (gopacket.Flow, error) {
	m := encodableFlow{}

	by, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return gopacket.Flow{}, err
	}
	b := bytes.Buffer{}
	b.Write(by)

	d := gob.NewDecoder(&b)
	err = d.Decode(&m)
	if err != nil {
		return gopacket.Flow{}, err
	}

	return gopacket.NewFlow(m.FlowType, m.FlowSrc, m.FlowDst), nil
}
