package main

import (
	"github.com/lucas-clemente/quic-go"
	"net"
	"strconv"
	"fmt"
	"quic_utils"
	"io"
	"github.com/lucas-clemente/quic-go/qerr"
)

type portForwardingSession struct {
	sshConfig       *SSHConfig
	QUICSession     quic.Session
	QUICFirstStream quic.Stream
	client          *clientServed
}

type portForwardingFlow struct {
	initialConfig *portForwardingSession
	TCPConnection net.Conn
	QUICStream    quic.Stream
	localPort     uint16
	remotePort    uint16
	remoteIP      []byte
}

func newPortForwardingSession(sshConfig *SSHConfig, session quic.Session, firstStream quic.Stream) (*portForwardingSession) {
	return &portForwardingSession{
		sshConfig:       sshConfig,
		QUICSession:     session,
		QUICFirstStream: firstStream,
	}
}

func (pFSession *portForwardingSession) setClientServed(client *clientServed) {
	pFSession.client = client
}

// assuming port forwarding forwards payloads from an intermediate A to another intermediate B,
// runAsSource launch port forwarding for A. (Thus it listens on socket and forward payloads to B)
func (pFSession *portForwardingSession) runAsSource(localPort uint16, remotePort uint16, remoteIP []byte) {

	// step 1) open local TCPListener (socket TCPListener)
	TCPListener := pFSession.acceptLocalConnection(localPort)
	if TCPListener == nil {
		writeError(pFSession, nil,"Maybe chosen port is already used")
		return
	}

	for {
		// step 2) accept connections on the TCPListener
		TCPConnection, err := TCPListener.Accept()
		if err != nil {
			// if err != nil , stop listening. This can be because QUICSession was closed and thus we closed the TCPListener.
			return
		}
		go func() {
			// step 3) open a new QUICStream with port forwarding destination
			stream, err := pFSession.QUICSession.OpenStreamSync()
			if err != nil {
				writeError(pFSession, nil,"Cannot open stream")
				TCPConnection.Close()
				return
			}

			// step 4) create final port forwarding config
			forwardingConfig := pFSession.newPortForwardingFlow(TCPConnection, stream, localPort, remotePort, remoteIP)

			// step 5) Tell destination which hostname and port it must take through a well defined control message
			err = writeControlMessage(stream, true, localPort, remotePort, remoteIP)
			if err != nil {
					writeError(pFSession, stream,"Problem when writing on stream")
					TCPConnection.Close()
					return
			}

			// step 6) send and receive data from TCPConnection/QUICStream to QUICStream/TCPConnection
			finish1 := make(chan bool)
			finish2 := make(chan bool)
			go forwardingConfig.readQuicSendTCP(finish1)
			go forwardingConfig.readTCPSendQUIC(finish2)
			select { // wait that transmissions are finished on both QUICStream and local TCPConnection
			case <-finish1:
				<-finish2
			case <-finish2:
				<-finish1
			}
		}()

	}
}

// assuming port forwarding forwards payloads from an intermediate A to another intermediate B,
// runAsDestination launch port forwarding for B. (Thus it receives payloads from A and forwards them
// to final destination + listen answers and forward them to A.)
func (pFSession *portForwardingSession) runAsDestination() {
	for{
		// step 1) accept a QUICStream
		QUICStream, err := pFSession.QUICSession.AcceptStream()
		if err != nil {
			quicErr := qerr.ToQuicError(err)
			if quicErr.ErrorCode == qerr.PeerGoingAway || quicErr.ErrorCode == qerr.NetworkIdleTimeout {
				// this is normal if the QUICStream was closed so just exit this function
				return
			} else {
				writeError(pFSession, nil,"Additional stream cannot be opened")
			}
		}

		go func() {
			// step 2) read control message
			err, local, localPort, remotePort, remoteIP := pFSession.readControlMessage(QUICStream)
			if err != nil {
				writeError(pFSession, QUICStream,"A problem appeared when reading control informations about port forwarding.")
				return
			}else if(pFSession.sshConfig.listen){
				// below: comment or uncomment to see port forwarding requests on server side
				pFSession.sshConfig.printDebug(fmt.Sprintf("New forwarding: %d:[%s]:%d (StreamID=%d)", localPort, ipToString(remoteIP), remotePort, QUICStream.StreamID()))
			}

			// step 3) [Optional] if local=false, then client asks for "remote" port forwarding so we must ask as source
			if !local {
				QUICStream.Close() // in this particular case the QUICStream was just used to ask the remote port forwarding
				pFSession.runAsSource(localPort, remotePort, remoteIP)
				return
			}

			// step 4) Contact remoteIP
			TCPConn, err := net.Dial("tcp", ipToString(remoteIP)+":"+strconv.Itoa(int(remotePort)))
			if err != nil {
				QUICStream.Close()
				return
			}

			// step 5) create final port forwarding config
			forwardingConfig := pFSession.newPortForwardingFlow(TCPConn, QUICStream, localPort, remotePort, remoteIP)

			// step 6) send and receive data from connection/QUICStream to QUICStream/connection
			finishQuicStreamToTCP := make(chan bool) // this channel is used to indicate when the QUIC stream seems closed when trying to read on it.
			finishTCPToQUICStream := make(chan bool) // this channel is used to indicate when the TCP connection seems closed when reading on it.

			go forwardingConfig.readQuicSendTCP(finishQuicStreamToTCP)
			go forwardingConfig.readTCPSendQUIC(finishTCPToQUICStream)

			//below: wait for both 'finish channels'
			select {
			case <-finishQuicStreamToTCP:
				<-finishTCPToQUICStream
			case <-finishTCPToQUICStream:
				<-finishQuicStreamToTCP
			}
		}()
	}
}

func (pFSession *portForwardingSession) acceptLocalConnection(localPort uint16) (net.Listener) {
	portStr := ":" + strconv.Itoa(int(localPort)) //"127.0.0.1:" + strconv.Itoa(int(localPort))
	listener, err := net.Listen("tcp", portStr)
	if pFSession.sshConfig.listen {
		if err != nil { // if i am server, do not crash because of bad client request
			return nil
		} else { // register which client asked for this remote port forwarding to stop listening when connection finished
			addr := fmt.Sprintf("%s", pFSession.QUICSession.RemoteAddr())
			pFSession.client.listActiveListeners[addr] = append(pFSession.client.listActiveListeners[addr], listener)
		}
	} else {
		quic_utils.Check(err)
	}
	return listener
}

// if error appear on server side, stop the port forwarding without crashing the server
func writeError(pFSession *portForwardingSession, forwardingStream quic.Stream, msg string) {
	if pFSession.sshConfig.listen{
		toSend := []byte(fmt.Sprintf("Error with port forwarding. %s\n", msg))
		pFSession.QUICFirstStream.Write(toSend)
	}else{
		fmt.Printf("Error with port forwarding. %s\n", msg)
	}
	if forwardingStream != nil{
		forwardingStream.Close()
	}
}

func (pFSession *portForwardingSession) newPortForwardingFlow(conn net.Conn, stream quic.Stream, localPort uint16, remotePort uint16, remoteIP []byte) (*portForwardingFlow) {
	return &portForwardingFlow{
		initialConfig: pFSession,
		TCPConnection: conn,
		QUICStream:    stream,
		localPort:     localPort,
		remotePort:    remotePort,
		remoteIP:      remoteIP,
	}
}

// Reads msg from TCP connection and forwards payloads on QUIC stream.
// If TCP connection is detected closed, close the stream and stop this method.
func (pFFlow *portForwardingFlow) readTCPSendQUIC(finish chan bool) {
	bufSize := pFFlow.initialConfig.sshConfig.bufSize
	readBuffer := make([]byte, bufSize, bufSize)
	for {
		n, err := io.ReadAtLeast(pFFlow.TCPConnection, readBuffer, 1)
		if err != nil {
			pFFlow.QUICStream.Close()
			finish <- true
			return
		} else {
			msg := readBuffer[:n]
			n2, err2 := pFFlow.QUICStream.Write(msg)
			if n2 != n || err2 != nil {
				pFFlow.QUICStream.Close()
				finish <- true
				return
			}
		}
	}
}

// Reads payloads from QUIC stream and forwards them on TCP connection.
// If QUIC stream is detected closed, close the TCP connection and stop this method.
func (pFFlow *portForwardingFlow) readQuicSendTCP(finish chan bool) {
	bufSize := pFFlow.initialConfig.sshConfig.bufSize
	readBuffer := make([]byte, bufSize, bufSize)
	for {
		n, err := pFFlow.QUICStream.Read(readBuffer)
		if err == nil || n > 0 {
			msg := readBuffer[:n]
			n2, err2 := pFFlow.TCPConnection.Write(msg)
			if n != n2 || err2 != nil {
				pFFlow.TCPConnection.Close()
				finish <- true
				return
			}
		}
		if err != nil {
			pFFlow.TCPConnection.Close()
			finish <- true
			return
		}
	}
}
