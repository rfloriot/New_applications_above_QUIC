package main

import (
	"github.com/lucas-clemente/quic-go"
	"encoding/binary"
	"io"
	"net"
	"errors"
)

/*
    Format for control messages:
    ----------------------------

	Arbitrary control message use TLV encoding:

	0       8       16
    +-+-+-+-+-+-+-+-+-+-+-+-+---
    | type  |length | value
    +-+-+-+-+-+-+-+-+-+-+-+-+---

	Possible types are:
    > 0x01 for "local port forwarding request",
    > 0x02 for "remote port forwarding request".

	Below, we detail the local and remote port forwarding request message:

	1) local port forwarding request:

    0       8       16             31
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
    |t=0x01 |l=0xF3 |  remote port  |
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
    | prot. |                       |
	+-------+						+
	|								|
    .			remote IP 			.
    .			(16 bytes)			.
	|                               |
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+



	2) remote port forwarding request:

    0       8       16             31
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
    |t=0x02 |l=0xF5 |  local port   |
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+
    |  remote port  | prot. |       |
    +-+-+-+-+-+-+-+-+-+-+-+-+       +
	|                               |
    .			remote IP 			.
    .			(16 bytes)			.
	|                               |
    +-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+-+


	fields:
	-------
    remote port: The receiver of this control message will use it for forwarding message. Encoded on 2 bytes so limited to the range [0, 65535]
	local port : The receiver of this control message will use it for listening new connections. Encoded on 2 bytes so limited to the range [0, 65535]
	prot.      : The protocol to forward (0x06 for TCP).
	remote IP  : Final destination of port forwarding. Encoded as IPv6, it can handle IPv4 too by using a fixed IPv6 prefix : 64:ff9b::/96 (RFC 6052)

	Note: remote port forwarding implementation is very simple and understandable by just relying on local port forwarding implementation.
    The "cost" of this simplicity is to transmit remotePort & remoteIP inside remote port forwarding control message while it could not be transferred.

*/

/*
 * read control message on stream following schema depicted above.
 */
func (pFSession *portForwardingSession) readControlMessage(stream quic.Stream) (err error, isLocalPortForwarding bool, localPort uint16, remotePort uint16, remoteIP net.IP) {

	typeBuffer := make([]byte, 1, 1)
	lengthBuffer := make([]byte, 1, 1)
	localPortBuffer := make([]byte, 2, 2)
	remotePortBuffer := make([]byte, 2, 2)
	protocolBuffer := make([]byte, 1, 1)
	var remoteIPBuffer net.IP = make([]byte, net.IPv6len, net.IPv6len)

	// read type field
	n, err := io.ReadFull(stream, typeBuffer)
	if err != nil || n != 1 {
		err = errors.New("error when reading stream")
		return
	}
	localValue := uint8(typeBuffer[0])
	if localValue == 0x01 {
		isLocalPortForwarding = true
	} else if localValue == 0x02 {
		isLocalPortForwarding = false
	} else {
		err = errors.New("error with the values read on the stream")
		return
	}

	// read length field
	n, err = io.ReadFull(stream, lengthBuffer)
	if err != nil || n != 1 {
		err = errors.New("error when reading stream")
		return
	}
	lengthValue := uint8(lengthBuffer[0])
	if (lengthValue != 19 && isLocalPortForwarding) || (lengthValue != 21 && !isLocalPortForwarding) {
		err = errors.New("error with the values read on the stream")
	}

	// read port(s)
	if isLocalPortForwarding {
		n, err = io.ReadFull(stream, remotePortBuffer)
		if err != nil || n != 2 {
			err = errors.New("error when reading stream")
			return
		}
		remotePort = binary.BigEndian.Uint16(remotePortBuffer)
	} else {
		n, err = io.ReadFull(stream, localPortBuffer)
		if err != nil || n != 2 {
			err = errors.New("error when reading stream")
			return
		}
		localPort = binary.BigEndian.Uint16(localPortBuffer)

		n, err = io.ReadFull(stream, remotePortBuffer)
		if err != nil || n != 2 {
			err = errors.New("error when reading stream")
			return
		}
		remotePort = binary.BigEndian.Uint16(remotePortBuffer)
	}

	// read protocol number (nothing done with it for now, only tcp is used).
	n, err = io.ReadFull(stream, protocolBuffer)
	if err != nil || n != 1 {
		err = errors.New("error when reading stream")
		return
	}

	// read remote ip
	n, err = io.ReadFull(stream, remoteIPBuffer)
	if err != nil || n != len(remoteIPBuffer) {
		err = errors.New("error when reading stream")
		return
	}
	if isV4EncodedInV6(remoteIPBuffer[:n]){
		remoteIP = getV4FromV6(remoteIPBuffer[:n])
	}else{
		remoteIP = remoteIPBuffer[:n]
	}
	err = nil
	return
}

// does ipv6 begin with 64:ff9b::/96 ?
var V4_TO_V6_PREFIX = [...]byte {0, 100, 255, 155, 0, 0, 0, 0, 0, 0, 0, 0}
func isV4EncodedInV6(remoteIP net.IP) bool {
	for i:= 0; i < 12; i++{
		if V4_TO_V6_PREFIX[i] != remoteIP[i]{
			return false
		}
	}
	return true
}

func getV4FromV6(remoteIP net.IP) (result net.IP){
	result = make([]byte, net.IPv4len, net.IPv4len)
	for i, j:= 12, 0; i < 16; i, j = i+1, j+1{
		result[j] = remoteIP[i]
	}
	return result
}

func ipv4to6(remoteIP net.IP) (result net.IP){
	result = make([]byte, net.IPv6len, net.IPv6len)
	for i:= 0; i < 12; i++{
		result[i] = V4_TO_V6_PREFIX[i]
	}
	for i, j:= 12, 0; i < 16; i, j = i+1, j+1{
		result[i] = remoteIP[j]
	}
	return result
}

/*
 * write control message on stream following schema depicted above.
 */
func writeControlMessage(stream quic.Stream, local bool, localPort uint16, remotePort uint16, remoteIP net.IP) (err error) {
	typeBuffer := make([]byte, 1, 1)
	lengthBuffer := make([]byte, 1, 1)
	locPoBuffer := make([]byte, 2, 2)
	remPoBuffer := make([]byte, 2, 2)
	protocolBuffer := make([]byte, 1, 1)
	var remoteIPV6 net.IP
	if local {
		typeBuffer[0] = 0x01
		lengthBuffer[0] = 19
	} else {
		typeBuffer[0] = 0x02
		lengthBuffer[0] = 21
	}
	binary.BigEndian.PutUint16(locPoBuffer, localPort)
	binary.BigEndian.PutUint16(remPoBuffer, remotePort)

	protocolBuffer[0] = 0x06

	if len(remoteIP) == net.IPv4len {
		remoteIPV6 = ipv4to6(remoteIP)
	} else if len(remoteIP) == net.IPv6len {
		remoteIPV6 = remoteIP
	} else {
		return errors.New("error with the values passed in argument (len of remoteIP not consistent)")
	}

	buf := append(typeBuffer, lengthBuffer...)
	if(local){
		buf = append(buf, remPoBuffer...)
	}else{
		buf = append(buf, locPoBuffer...)
		buf = append(buf, remPoBuffer...)
	}

	buf = append(buf, protocolBuffer...)
	buf = append(buf, remoteIPV6...)
	n, err := stream.Write(buf)
	if err != nil || (n != 21 && local) || (n != 23 && !local) {
		return errors.New("error when writing on the stream")
	}

	return nil
}
