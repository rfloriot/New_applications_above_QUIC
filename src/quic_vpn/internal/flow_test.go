package internal

import (
	"github.com/google/gopacket"
	"testing"
)

// Test find + encode-decode valid packets
func TestFlow_ValidPackets(t *testing.T) {
	testData := []struct {
		packet     []byte
		flowString string
	}{
		{mockTcpPacket, mockTcpPacketFlow},
		{mockTcpPacket2, mockTcpPacketFlow2},
		{mockPingPacket, mockPingPacketFlow},
	}

	for _, data := range testData {
		flow := testFindFlow(data, t)
		testEncodeDecode(flow, t)
	}
}

func testFindFlow(data struct {
	packet     []byte
	flowString string
}, t *testing.T) gopacket.Flow {

	flow, err := FindFlow(data.packet)

	if err != nil {
		t.Errorf("Unable to find flow: %v\n", err)
	}

	if flow.String() != data.flowString {
		t.Errorf("Invalid flow conversion (%v; %v expected)", flow.String(), data.flowString)
	}

	return flow
}

func testEncodeDecode(flow gopacket.Flow, t *testing.T) {
	encoded := EncodeFlow(flow)
	decoded, err := DecodeFlow(encoded)

	if err != nil {
		t.Errorf("Unable to decode flow: %v\n", err)
	}

	if decoded != flow {
		t.Errorf("Invalid flow (encoded %v != decoded %v)", flow, decoded)
	}
}

// Test find invalid packet
func TestFlow_InvalidPacket(t *testing.T) {
	invalidIPpacket := []byte{0xab, 0xbc}

	_, err := FindFlow(invalidIPpacket)
	if err == nil {
		t.Errorf("Unable to detect invalid flow from %v\n", invalidIPpacket)
	}
}

// Test decode invalid strings
func TestFlow_InvalidCoding(t *testing.T) {

	_, err := DecodeFlow("INVALID")
	if err == nil {
		t.Errorf("Unable to detect invalid flow from %s\n", "INVALID")
	}

	_, err = DecodeFlow("Y291Y291")
	if err == nil {
		t.Errorf("Unable to detect invalid flow from %s\n", "INVALID")
	}
}
