package internal

import (
	"testing"
	"encoding/binary"
)


func TestMarkECN(t *testing.T) {
	v4packet := make([]byte, len(mockPingPacket))
	copy(v4packet, mockPingPacket)

	MarkECN(v4packet)
	computedCheckSum := recomputeV4Checksum(mockPingPacket)

	if computedCheckSum != 1752 {
		t.Errorf("Invalid checksum of ECN marked packet %v != 1749 (hand computed)\n", computedCheckSum);
	}

	if v4packet[1] & 0x3 != 3 {
		t.Errorf("Packet non ECN marked!\n");
	}

	v6packet := make([]byte, len(mockTcpPacket))
	copy(v6packet, mockTcpPacket)
	MarkECN(v6packet)

	if v6packet[1] & 0x30 == 0 {
		t.Errorf("Packet non ECN marked!\n");
	}
}


func TestComputeChecksum(t *testing.T) {
	observedChecksum := binary.BigEndian.Uint16(mockPingPacket[10:12])
	computedChecksum := recomputeV4Checksum(mockPingPacket)
	if observedChecksum != computedChecksum {
		t.Errorf("Invalid checksum recomputation %v != %v\n", observedChecksum, computedChecksum)
	}
}
