// Datagram mode over a QUIC byte stream
// By CLAREMBEAU Alexis & FLORIOT Remi
// 2017-2018 Master's thesis. All rights reserved.

package internal

import (
	"encoding/binary"
	"github.com/lucas-clemente/quic-go"
	"io"
)

type Datagram struct {
	Payload []byte
}

// Send a datagram to a quic stream
func (d *Datagram) Send(stream quic.Stream) error {
	if len(d.Payload) == 0 {
		return nil
	}

	// Send size
	sizeBuffer := make([]byte, 4, 4)
	binary.BigEndian.PutUint32(sizeBuffer, uint32(len(d.Payload)))

	// Send payload
	if len(d.Payload) != 0 {
		_, err := stream.Write(append(sizeBuffer,d.Payload...))
		if err != nil {
			return err
		}
	}

	return nil
}

// Read a datagram from a quic stream
func Recv(stream quic.Stream) (Datagram, error) {
	// Read size
	sizeBuffer := make([]byte, 4, 4)
	_, err := io.ReadFull(stream, sizeBuffer)
	if err != nil {
		return Datagram{}, err
	}
	size := binary.BigEndian.Uint32(sizeBuffer)

	// Read payload 
	byteBuffer := make([]byte, size, size)
	if size > 0 {
		_, err = io.ReadFull(stream, byteBuffer)
		if err != nil {
			return Datagram{}, err
		}
	}
	return Datagram{Payload: byteBuffer}, nil

}
