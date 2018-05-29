// VPN : Client mode
// By CLAREMBEAU Alexis & FLORIOT Remi
// 2017-2018 Master's thesis. All rights reserved.

package main

import (
	"crypto/rsa"
	"crypto/tls"
	"errors"
	"github.com/lucas-clemente/quic-go"
	"github.com/songgao/water"
	"quic_utils"
	. "quic_vpn/internal"
	"strconv"
)

// Structure representing the program in client mode
type ClientInstance struct {
	vpnConfig       *VpnConfig
	tunnelInterface *water.Interface
	session         quic.Session
	controlStream   quic.Stream
	lastError       error
}

// Start a new program in client mode
func NewClientInstance(config *VpnConfig) (*ClientInstance, error) {
	iface, err := NewTunnelInterface(config)
	if err != nil {
		return nil, err
	}

	return &ClientInstance{vpnConfig: config, tunnelInterface: iface}, nil
}

// Run the program in client mode
func (c *ClientInstance) Run() error {
	println("dial server")
	c.lastError = c.dial()

	println("open control stream")
	c.lastError = c.openControlStream()

	println("check server")
	c.lastError = c.authenticateServer()



	if c.lastError != nil {
		return c.lastError
	}

	println("main loop")
	t := NewTransmitter(c.vpnConfig, c.session, c.tunnelInterface)
	return t.WaitOutput()
}

// Dial distant server
func (c *ClientInstance) dial() error {
	if c.lastError != nil {
		return c.lastError
	}

	dialedAddress := c.vpnConfig.Server.Addr + ":" + strconv.Itoa(c.vpnConfig.Server.Port)

	// extract keys
	publicKey, err := quic_utils.ExtractPublicKey(c.vpnConfig.Client.Public)
	if err != nil {
		return err
	}

	privateKey, err := quic_utils.ExtractPrivateKey(c.vpnConfig.Client.Private)
	if err != nil {
		return err
	}

	cert, err := quic_utils.MakeCertificate(publicKey, privateKey)
	if err != nil {
		return err
	}

	// dial
	session, err := quic.DialAddr(
		dialedAddress,
		&tls.Config{
			InsecureSkipVerify: true,
			Certificates: []tls.Certificate{cert},
		},
		&quic.Config{
			KeepAlive: true,
		},
	)
	c.session = session
	return err
}

// Check the authenticity given by the server
func (c *ClientInstance) authenticateServer() error {
	if c.lastError != nil {
		return c.lastError
	}

	if c.vpnConfig.Server.Check_key {
		// check server private key
		expectedKey, err := quic_utils.ExtractPublicKey(c.vpnConfig.Server.Public)
		if err != nil {
			return err
		}

		serverKey := c.session.ConnectionState().PeerCertificates[0].PublicKey.(*rsa.PublicKey)

		if !quic_utils.ComparePublicKeys(expectedKey, serverKey) {
			err := errors.New("key verification failed")
			c.session.Close(err)
			return err
		}

		// serve client public key
		clientPublicKey, err := quic_utils.ExtractPublicKey(c.vpnConfig.Client.Public)
		quic_utils.Check(err)
		clientPrivateKey, err := quic_utils.ExtractPrivateKey(c.vpnConfig.Client.Private)
		quic_utils.Check(err)

		quic_utils.ServeClientPublicKey(c.session, c.controlStream, clientPrivateKey, clientPublicKey)

		return nil
	}
	return nil
}

// Open a new control stream (to serve client key)
func (c *ClientInstance) openControlStream() error {
	if c.lastError != nil {
		return c.lastError
	}

	controlStream, err := c.session.OpenStreamSync()
	c.controlStream = controlStream
	return err
}
