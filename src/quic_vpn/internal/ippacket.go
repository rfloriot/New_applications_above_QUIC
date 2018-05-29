package internal

import (
	"math/big"
	"encoding/binary"
)

func MarkECN(packet []byte) {
	// add congestion
	if packet[0]>>4 == 4 {
		packet[1] |= 0x3
		checksum := recomputeV4Checksum(packet)
		binary.BigEndian.PutUint16(packet[10:12], checksum)

	} else if packet[0]>>4 == 6 {
		packet[1] |= 0x30
	} else {
	}
}

func MarkECE(packet []byte) {
	if len(packet) > 0 && packet[0] >>4 == 4 {
		tcpIndex := packet[0] & 0x0f * 4;
		flagsIndex := tcpIndex + 3*4  +1;
		packet[flagsIndex] |= 0x40; // set ECE bit
	}
}


func And(x, y *big.Int) *big.Int {
	return big.NewInt(0).And(x, y)
}
func Add(x, y *big.Int) *big.Int {
	return big.NewInt(0).Add(x, y)
}
func Rsh(x *big.Int, y uint) *big.Int {
	return big.NewInt(0).Rsh(x, y)
}

func recomputeV4Checksum(packet []byte) uint16{
	sum := big.NewInt(0)

	words := [][]byte{
		packet[0:2],
		packet[2:4],
		packet[4:6],
		packet[6:8],
		packet[8:10],
		// packet[10:12], ; skip checksum
		packet[12:14],
		packet[14:16],
		packet[16:18],
		packet[18:20],
	}

	for _, w := range(words){
		intval := binary.BigEndian.Uint16(w)
		sum = Add(sum, big.NewInt(int64(intval)))
	}

	for Rsh(sum, 16).Uint64() > 0 { // while (sum >> 64) > 0
		carry := Rsh(sum, 16)
		rest := And(sum, big.NewInt(0xffff))

		sum = Add(carry, rest)
	}


	isum := uint16(sum.Uint64())
	return ^isum
}