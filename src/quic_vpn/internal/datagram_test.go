package internal

import (
	"bytes"
	"testing"
)

// test sending a void datagram
func TestDatagram_SendVoid(t *testing.T) {
	voidDatagram := Datagram{}
	err := voidDatagram.Send(nil)
	if err != nil {
		t.Errorf("Error sending void Datagram: %v", err)
	}
}

// test transmitting & receiving different datagrams
func TestDatagram_Transfer(t *testing.T) {
	cliSess, cliStream, servSess, servStream, err := MockClientServer("localhost:4041")

	if err != nil {
		t.Fatalf("Unable to start ClientInstance or ServerInstance %v", err)
	}
	defer cliSess.Close(nil)
	defer servSess.Close(nil)

	// table driven test
	table := []Datagram{
		{Payload: []byte("Hello world")},
		{Payload: []byte{0}},
		{Payload: []byte{0, 0, 0}},
	}

	for _, d := range table {
		go func() {
			err := d.Send(cliStream)
			if err != nil {
				t.Error(err)
			}
		}()

		receivedDatagram := Datagram{}
		receivedDatagram, err := Recv(servStream)
		if err != nil {
			t.Error(err)
		}

		if bytes.Compare(receivedDatagram.Payload, d.Payload) != 0 {
			t.Errorf("(%v sent) != (%v recv)", d.Payload, receivedDatagram.Payload)
		}
	}
}
