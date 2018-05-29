package main

import (
	"github.com/lucas-clemente/quic-go"
	"os"
	"io"
	"fmt"
	"github.com/lucas-clemente/quic-go/qerr"
	"crypto/x509"
	"quic_utils"
	"errors"
	"crypto/tls"
	"crypto/rsa"
	"github.com/alexflint/go-arg"
	"time"
)

// ================== configuration, parsing from command line ==================

type cli_config struct {
	Listen  bool   `arg:"-l" help:"do we listen or connect? (default=no)"`
	Debug 	bool   `arg:"-d" help:"do we set debug mode? (default=no)"`
	Host    string `arg:"positional" help:"host to contact"`
	Port    int    `arg:"positional, required" help:"port to connect or listen"`
	PubKey  string `arg:"--pub" help:"server public key"`
	PrivKey string `arg:"--priv" help:"server private key"`
	ReqKey  string `arg:"--req" help:"remote host public key"`
	BufSize int    `arg:"-b" help:"internal buffer size (default=200000)"`
}

func (c *cli_config) formatAddress() string {
	return fmt.Sprintf("%v:%v", c.Host, c.Port)
}

func main() {
	conf := cli_config{BufSize:200000}
	p := arg.MustParse(&conf)
	if conf.Listen && (conf.PubKey == "" || conf.PrivKey == ""){
		p.Fail("you must provide public and private key in listen mode")
	}

	if conf.Listen {
		quic_utils.Check(runServer(&conf))
	} else {
		quic_utils.Check(runClient(&conf))
	}
}

// ==================================== server ====================================

func runServer(conf *cli_config) error {
	publicKey, err := quic_utils.ExtractPublicKey(conf.PubKey)
	privateKey, err := quic_utils.ExtractPrivateKey(conf.PrivKey)

	cert, err := quic_utils.MakeCertificate(publicKey, privateKey)
	quic_utils.Check(err)

	tlsConf := tls.Config{Certificates: []tls.Certificate{cert}}
	debug(conf, "cli_config generated")

	listener, err := quic.ListenAddr(conf.formatAddress(), &tlsConf, nil)
	quic_utils.Check(err)
	debug(conf, "address listened")

	session, err := listener.Accept()
	quic_utils.Check(err)
	debug(conf, "client accepted")

	stream, err := session.AcceptStream()
	quic_utils.Check(err)
	debug(conf, "stream accepted")

	if conf.ReqKey != "" {
		receivedKey, _ := quic_utils.AskClientPublicKey(stream)
		debug(conf, "key received")

		requiredClientKey, _ := quic_utils.ExtractPublicKey(conf.ReqKey)

		if !quic_utils.ComparePublicKeys(receivedKey, requiredClientKey) {
			session.Close(errors.New("client public key refused"))
			return errors.New("client public key refused")
		}

		debug(conf, "key accepted")
	}

	debug(conf, "communication ready!")
	return loop(conf, session, stream)
}

// ==================================== client ====================================

func runClient(conf *cli_config) error {
	session, err := quic.DialAddr(conf.formatAddress(), &tls.Config{InsecureSkipVerify: true}, nil)
	quic_utils.Check(err)
	debug(conf, "server contacted")

	// check handshake server side public key
	if conf.ReqKey != "" {
		requiredServerKey, err := quic_utils.ExtractPublicKey(conf.ReqKey)
		quic_utils.Check(err)

		cert, err := x509.ParseCertificate(session.AddedForThesis_getLeafCert())
		quic_utils.Check(err)
		debug(conf, "server key received")

		if !quic_utils.ComparePublicKeys(cert.PublicKey.(*rsa.PublicKey), requiredServerKey) {
			session.Close(errors.New("server public key refused"))
			return errors.New("server public key refused")
		} else {
			debug(conf, "server key accepted")
		}
	}

	stream, err := session.OpenStreamSync()
	quic_utils.Check(err)
	debug(conf, "stream opened")

	if conf.ReqKey != "" {
		publicKey, err := quic_utils.ExtractPublicKey(conf.PubKey)
		quic_utils.Check(err)
		privateKey, err := quic_utils.ExtractPrivateKey(conf.PrivKey)
		quic_utils.Check(err)

		quic_utils.ServeClientPublicKey(stream, privateKey, publicKey)
		debug(conf, "client key served")
	}

	debug(conf, "communication ready!")
	return loop(conf, session, stream)
}

// ============================== core transmission system ==============================

// main loop subfunction
func loop(conf *cli_config, session quic.Session, stream quic.Stream) error {
	errorChannel := make(chan error)

	go transmit(conf, errorChannel, stream, os.Stdout)
	go transmit(conf, errorChannel, os.Stdin, stream)

	err := <-errorChannel
	quicErr := qerr.ToQuicError(err)

	if err == io.EOF {
		session.Close(nil)
		return nil
	} else if quicErr.ErrorCode == qerr.PeerGoingAway {
		return nil
	} else {
		return err
	}
}

// transmit from in to out, signaling errors to communicationChannel
func transmit(conf *cli_config, communicationChannel chan error, in io.Reader, out io.Writer) {
	buffer := make([]byte, conf.BufSize, conf.BufSize)
	for {
		n, err := in.Read(buffer)
		if err != nil {
			communicationChannel <- err
			return
		}

		out.Write(buffer[:n])
	}
}

func debug(conf *cli_config, format string, args ...interface{}){
	if conf.Debug {
		fmt.Printf("[debug] " + format + "\n", args...)
	}
}