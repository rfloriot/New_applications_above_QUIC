package main

import (
	"github.com/lucas-clemente/quic-go"
	"crypto/x509"
	"crypto/tls"
)

func (config *SSHConfig) getServerCert(session quic.Session) *x509.Certificate {
	// single path version:
	config.printDebug("This is single path mode")
	cert := session.ConnectionState().PeerCertificates[0]

	// multi path version:
	// config.printDebug("This is multi path mode")
	// cert, err := x509.ParseCertificate(session.AddedForThesis_getLeafCert())
	// quic_utils.Check(err)
	return cert
}

func (config *SSHConfig) openSession() (quic.Session, error){
	return quic.DialAddr(config.formatAddress(), &tls.Config{InsecureSkipVerify: true}, &quic.Config{KeepAlive: true}, )
}