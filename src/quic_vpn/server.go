// VPN: Server mode
// By CLAREMBEAU Alexis & FLORIOT Remi
// 2017-2018 Master's thesis. All rights reserved.
package main

import (
	"crypto/tls"
	"errors"
	"github.com/lucas-clemente/quic-go"
	"github.com/songgao/water"
	"quic_utils"
	. "quic_vpn/internal"
	"strconv"
)

// Structure representing the program in server mode
type ServerInstance struct {
	vpnConfig       *VpnConfig
	tunnelInterface *water.Interface
	tlsConfig       *tls.Config
	listener        quic.Listener
}

// Start a new program in server mode
func NewServerInstance(config *VpnConfig) (*ServerInstance, error) {
	iface, err := NewTunnelInterface(config)
	if err != nil {
		return nil, err
	}

	return &ServerInstance{
		vpnConfig:       config,
		tunnelInterface: iface,
	}, nil
}

// Run the program in server mode
func (s *ServerInstance) Run() error {
	println("init TLS config")
	if err := s.initTlsConfig(); err != nil {
		return err
	}

	println("listen")
	if err := s.listen(); err != nil {
		return err
	}

	println("wait clients")
	for {
		session, err := s.listener.Accept()
		if err != nil {
			return err
		}

		println("    new client")
		cli := connectedClient{server: s, session: session, vpnConfig: s.vpnConfig}
		go cli.Handle()
	}

	return nil
}

// Initialize QUIC tls config
func (s *ServerInstance) initTlsConfig() error {
	publicKey, err := quic_utils.ExtractPublicKey(s.vpnConfig.Server.Public)
	if err != nil {
		return err
	}

	privateKey, err := quic_utils.ExtractPrivateKey(s.vpnConfig.Server.Private)
	if err != nil {
		return err
	}

	cert, err := quic_utils.MakeCertificate(publicKey, privateKey)
	if err != nil {
		return err
	}

	s.tlsConfig = &tls.Config{Certificates: []tls.Certificate{cert}}
	return nil
}

// Listen for new incomming connections
func (s *ServerInstance) listen() error {
	listenedAddress := s.vpnConfig.Server.Addr + ":" + strconv.Itoa(s.vpnConfig.Server.Port)
	listener, err := quic.ListenAddr(listenedAddress, s.tlsConfig, &quic.Config{})
	if err != nil {
		return err
	}
	s.listener = listener
	return nil
}

// Structure representing a connected client
type connectedClient struct {
	server        *ServerInstance
	session       quic.Session
	controlStream quic.Stream
	vpnConfig     *VpnConfig
}

// Handle a new connected client (wait & serve)
func (t *connectedClient) Handle() error {

	if err := t.waitControlStream(); err != nil {
		return err
	}

	if err := t.checkAuthenticity(); err != nil {
		return err
	}

	tr := NewTransmitter(t.vpnConfig, t.session, t.server.tunnelInterface)
	return tr.WaitOutput()
}

// Wait the first control stream from the client
func (t *connectedClient) waitControlStream() error {
	controlStream, err := t.session.AcceptStream()
	t.controlStream = controlStream
	return err
}

// Check client authenticity (if required)
func (t *connectedClient) checkAuthenticity() error {
	if t.server.vpnConfig.Client.Check_key {
		expectedKey, err := quic_utils.ExtractPublicKey(t.server.vpnConfig.Client.Public)
		if err != nil {
			return err
		}


		clientKey, err := quic_utils.AskClientPublicKey(t.session, t.controlStream)

		if err != nil && !quic_utils.ComparePublicKeys(expectedKey, clientKey) {
			err := errors.New("server thread: expected key != received key")
			t.session.Close(err)
			return err
		}
	}
	return nil
}
