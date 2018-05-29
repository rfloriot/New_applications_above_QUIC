package internal

import (
	"github.com/songgao/water"
	"github.com/lucas-clemente/quic-go"
	"sync"
	"time"
	"github.com/google/gopacket"
	"errors"
	"fmt"
	"io"
	"sort"
)

type toSend struct {
	flow   gopacket.Flow
	packet []byte
	time   time.Time
}

type Transmitter struct {
	vpnConfig       *VpnConfig
	quicSession     quic.Session
	tunnelInterface *water.Interface
	lastError       chan error

	mapInteraction sync.Map
	mapQuicStream  sync.Map

	toSendQueue chan toSend
}

// Create a new transmission system
func NewTransmitter(vpnConfig *VpnConfig, session quic.Session, iface *water.Interface) *Transmitter {
	return &Transmitter{
		vpnConfig:       vpnConfig,
		quicSession:     session,
		tunnelInterface: iface,
		lastError:       make(chan error),
		toSendQueue:     make(chan toSend, 1000),
	}
}

func (t *Transmitter) WaitOutput() (error) {
	go t.ListenTun()
	go t.ListenNet()
	go t.CollectUnused()
	go t.SchedulePackets()


	return <-t.lastError
}

func (t *Transmitter) SchedulePackets(){
	for {
		top := <-t.toSendQueue

		if len(t.toSendQueue) == 0 {
			// only one packet to send: send
			t.sendPacket(top)
		} else{
			// otherwise: put all packets in a slice
			n := len(t.toSendQueue)
			workingSlice := make([]toSend, n+1)

			workingSlice[0] = top;
			for i := 0 ; i < n; i++ {
				workingSlice[i+1] = <- t.toSendQueue
			}

			// sort to prioritize (important to be stable to avoid reordering)
			sort.SliceStable(workingSlice, func (i, j int) bool {
				if t.vpnConfig.Mode == "client" {
					return workingSlice[i].flow.Src().LessThan(workingSlice[j].flow.Src());
				} else{
					return workingSlice[i].flow.Dst().LessThan(workingSlice[j].flow.Dst());
				}
			})


            t.sendPacket(workingSlice[0])
			for i, val := range(workingSlice){
                if(i>0){
				    t.toSendQueue <- val; 
                }
			}
		}
	}
}

func (t *Transmitter) sendPacket(p toSend){
	if time.Since(p.time) > t.quicSession.AddedForThesis_getRtt() {
		MarkECN(p.packet)
	}

	tmp, ok := t.mapQuicStream.Load(p.flow)
	if ok {
		stream := tmp.(quic.Stream)
		data := Datagram{Payload: p.packet}
		data.Send(stream)
	}
}

func (t *Transmitter) ListenTun() { // interface to network
	for {
		// 1. read packet
		packetBuf := make([]byte, readBufSize, readBufSize)
		readSize, err := t.tunnelInterface.Read(packetBuf)
		if err != nil {
			t.lastError <- err
			return
		}

		// 2. find flow
		flow, err := FindFlow(packetBuf[:readSize])
		if err != nil {
			t.lastError <- err
			return
		}
		if !t.vpnConfig.Multi_streams {
			flow = gopacket.Flow{}
		}

		// 3. if new: open
		_, found := t.mapInteraction.Load(flow)
		if !found {
			fmt.Printf("Open stream for flow: %v\n", flow)
			newStream, err := t.quicSession.OpenStream()

			if err != nil {
				t.lastError <- err
				return;
			}
			t.mapQuicStream.Store(flow, newStream)
		}

		t.mapInteraction.Store(flow, time.Now())

		// 4. send packet to network
		add := toSend{
			flow:   flow,
			packet: packetBuf[:readSize],
			time:   time.Now(),
		}
		t.toSendQueue <- add
	}
}

func (t *Transmitter) ListenNet() {
	for {
		// 1. wait stream
		stream, err := t.quicSession.AcceptStream()
		if stream == nil || err != nil {
			t.lastError <- err
			return
		}

		// 3. wait for data
		go t.ListenNet_handleStream(stream)
	}
}

func (t *Transmitter) ListenNet_handleStream(stream quic.Stream) {
	for {
		datagram, err := Recv(stream)
		if err == io.EOF {
			stream.Close()
			return
		} else if err != nil {
			t.lastError <- err
		}

		t.tunnelInterface.Write(datagram.Payload)
	}
}

func (t *Transmitter) CollectUnused() {
	for {
		if !t.vpnConfig.Multi_streams {
			return
		}

		// 1. find unused flow
		flow := gopacket.Flow{}
		found := false

		t.mapInteraction.Range(func(key, value interface{}) bool {
			if time.Since(value.(time.Time)) > inactivityTimeout {
				flow = key.(gopacket.Flow)
				found = true
				return false
			}
			return true
		})

		// 2. close it
		if found {
			fmt.Printf("Inactivity on flow: %v\n", flow)

			stream, ok := t.mapQuicStream.Load(flow)
			if !ok {

				t.lastError <- errors.New("Unable to load stream")
			} else {
				(stream.(quic.Stream)).Close()

				// Needed to close the flows
				b := make([]byte, 1, 1)
				(stream.(quic.Stream)).Read(b)

				// clean data
				t.mapQuicStream.Delete(flow)
				t.mapInteraction.Delete(flow)
			}
		} else {
			time.Sleep(inactivePollTime)
		}
	}
}
