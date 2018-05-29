package main

import (
	"github.com/lucas-clemente/quic-go"
	"quic_utils"
	"crypto/tls"
	"fmt"
	"github.com/lucas-clemente/quic-go/qerr"
	"io"
	"errors"
	"os"
)

type SSHServer struct {
	conf     *SSHConfig
	listener quic.Listener
}

type clientServed struct {
	session             quic.Session
	firstStream         quic.Stream
	stopSessionChannel  chan bool
	listActiveListeners map[string][]closable
}

const MODE_REM_LOGIN = 1
const MODE_PORT_FORW = 2
const MODE_BOTH = 3

func NewQuicSSHServer(config *SSHConfig) (*SSHServer) {
	// extract public and private keys from files and build certificates
	publicKey, err := quic_utils.ExtractPublicKey(config.pubKeyFile)
	quic_utils.Check(err)
	privateKey, err := quic_utils.ExtractPrivateKey(config.privKeyFile)
	quic_utils.Check(err)
	cert, err := quic_utils.MakeCertificate(publicKey, privateKey)
	quic_utils.Check(err)
	tlsConf := tls.Config{Certificates: []tls.Certificate{cert}}

	// creating listener to listen to clients when calling Run method
	listener, err := quic.ListenAddr(config.formatAddress(), &tlsConf, nil)
	quic_utils.Check(err)

	return &SSHServer{
		conf:     config,
		listener: listener,
	}
}

// Run the program in server mode. This allows multiple clients to connect simultaneously
// and it is decomposed in 7 steps as described inside the function.
func (s *SSHServer) Run() error {
	go runTelnetd()// launch telnetd for interactive session handling
	for {
		// Step 1) accept a new session
		client, err := s.acceptNewClient()
		s.conf.printDebug("New session opened")
		if err == nil {
			go func() {

				// Step 2) accept a new first stream for this session
				if s.acceptNewStream(client) != nil {
					client.session.Close(nil)
					return
				}
				s.conf.printDebug("New stream opened")

				// Step 3) authenticate and then allow or reject this client
				if !s.allowClient(client) {
					client.session.Close(errors.New("connection refused (public key not allowed)"))
					return
				}

				// Step 4) ask the server mode to the client (1 = only remote login, 2 = only port forwarding, 3 = both)
				err, serverMode := s.askServerMode(client)
				if err != nil {
					client.session.Close(nil)
					return
				}

				// Step 5) launch port forwarding and/or remote login.
				if serverMode == MODE_PORT_FORW || serverMode == MODE_BOTH {
					s.launchPortForwarding(client)
				}
				if serverMode == MODE_REM_LOGIN || serverMode == MODE_BOTH {
					s.launchRemoteLogin(client)
				}

				// Step 6) [optional] if MODE_PORT_FORW, listen on first stream for end of service request
				if serverMode == MODE_PORT_FORW {
					s.waitForClientStopRequest(client)
				}

				// Step 7) wait for message received on stopSessionChannel then close the session
				<-client.stopSessionChannel
				client.session.Close(nil)
				s.conf.printDebug("Connection closed with foreign host");
				stopListener(client)
			}()
		}
	}
	return nil
}

// accept a new session with a client.
func (s *SSHServer) acceptNewClient() (client *clientServed, err error) {
	session, err := s.listener.Accept()
	if err != nil {
		quicErr := qerr.ToQuicError(err)
		if quicErr.ErrorCode == qerr.PeerGoingAway || quicErr.ErrorCode == qerr.NetworkIdleTimeout {
			return nil, errors.New("normal error: it was just a client that leaves")
		} else {
			os.Exit(0);//just pressed Ctrl-c
		}
	}
	return &clientServed{
		session:             session,
		listActiveListeners: make(map[string][]closable),
		stopSessionChannel:  make(chan bool),
	}, nil
}

func (s *SSHServer) acceptNewStream(client *clientServed) (err error) {
	stream, err := client.session.AcceptStream()
	if err != nil {
		quicErr := qerr.ToQuicError(err)
		if quicErr.ErrorCode == qerr.PeerGoingAway || quicErr.ErrorCode == qerr.NetworkIdleTimeout {
			client.session.Close(nil)
			return errors.New("normal error: a client leaved and thus no stream can be accepted anymore for him")
		} else {
			quic_utils.Check(err)
		}
	}
	client.firstStream = stream
	return nil
}

func (s *SSHServer) allowClient(client *clientServed) (result bool) {
	receivedKey , err := quic_utils.AskClientPublicKey(client.session, client.firstStream)
	if err != nil{
		return false
	}
	return checkClientPublicKey(s, receivedKey)
}

/*
 * after getting allowed, first message the client sent is:
 * > "1" if the client wants remote login only
 * > "2" if the client wants port forwarding only
 * > "3" if the client wants both remote login and port forwarding
 * This method listen on the stream and return this number as an integer.
 */
func (s *SSHServer) askServerMode(client *clientServed) (err error, result int) {
	readBuffer := make([]byte, 1, 1)
	n, err := io.ReadFull(client.firstStream, readBuffer)
	if n == 1 {
		msg := string(readBuffer[:n])
		if msg == "1" {
			result = MODE_REM_LOGIN
		} else if msg == "2" {
			result = MODE_PORT_FORW
		} else if msg == "3" {
			result = MODE_BOTH
		} else {
			err = errors.New("bad server mode request")
		}
	}
	if err != nil {
		err = errors.New("error when reading server mode on stream")
	}
	return err, result
}

func (s *SSHServer) launchPortForwarding(client *clientServed) {
	initialForwardingConfig := newPortForwardingSession(s.conf, client.session, client.firstStream)
	initialForwardingConfig.setClientServed(client)
	go initialForwardingConfig.runAsDestination()
}

func (s *SSHServer) launchRemoteLogin(client *clientServed) {
	go remoteLoginServerLoops(client.firstStream, s, client.stopSessionChannel)
}

func (s *SSHServer) waitForClientStopRequest(client *clientServed) {
	go func() {
		stopBuffer := make([]byte, 4, 4) // stop message is "stop" (4 letters)
		io.ReadFull(client.firstStream, stopBuffer)
		s.conf.printDebug("Connection closed with foreign host");
		client.stopSessionChannel <- true // we anyway also stop if message is not "stop" because then something went wrong.
	}()
}

func stopListener(client *clientServed) {
	addr := fmt.Sprintf("%s", client.session.RemoteAddr())
	for _, element := range client.listActiveListeners[addr] {
		element.Close()
	}
	client.listActiveListeners[addr] = nil
}
