package main

import (
	"github.com/alexflint/go-arg"
	"fmt"
	"crypto/tls"
	"github.com/lucas-clemente/quic-go"
	"strconv"
	"strings"
	"quic_utils"
	"sync"
	"time"
)

const port = 8989
const bufSize = 100000

type config struct {
	Mode     string `arg:"positional, required" help:"Mode 	 (client or server)"`
	Endpoint string `arg:"-c" help:"Enpoint to connect 		 (client mode)"`
	Size     string `arg:"-s" help:"File size 				 (client mode)"`
	Public   string `arg:"--pub" help:"Public key 			 (server mode)"`
	Private  string `arg:"--priv" help:"Private key 			 (server mode)"`
	Port     int    `arg:"-p" help:"Port to use 			 (both modes)"`
	Streams  int    `arg:"-n" help:"Number of streams to use (client mode)"`
}

func main() {
	conf := config{
		Port: port,
	}
	arg.MustParse(&conf)

	if conf.Mode == "client" {
		fmt.Printf("Client mode\nEndpoint: %v\nSize: %v\n", conf.Endpoint, conf.Size)
		clientMain(&conf)
	} else {
		fmt.Printf("Server mode\n")
		serverMain(&conf)
	}
}

func clientMain(c *config) {
	session, err := quic.DialAddr(c.Endpoint+":"+strconv.Itoa(c.Port), &tls.Config{InsecureSkipVerify: true}, nil)
	quic_utils.Check(err)

	m := sync.Mutex{}
	numEnded := 0

	//t0 := time.Now()

	//fmt.Printf("%v start\n",  time.Now().Second()*1000+time.Now().Nanosecond()/1000000.0)

	for i := 0; i < c.Streams; i++ {
		go func() {
			stream, err := session.OpenStreamSync()
			quic_utils.Check(err)

			size := c.Size
			size = strings.Replace(size, "k", "000", 1)
			size = strings.Replace(size, "M", "000000", 1)

			intSize, err := strconv.ParseInt(size, 10, 0)
			quic_utils.Check(err)

			stream.Write([]byte(strconv.Itoa(int(intSize))))

			n := 0
			buf := make([]byte, bufSize, bufSize)
			for n < int(intSize) {
				add, err := stream.Read(buf)
				quic_utils.Check(err)
				n += add
			}

			m.Lock()
			numEnded += 1
			m.Unlock()
			//fmt.Printf("%v stream %v finished\n", time.Now().Second()*1000+time.Now().Nanosecond()/1000000.0,stream.StreamID())
		}()
	}

	for numEnded < c.Streams {
		time.Sleep(5 * time.Millisecond)
	}

	//fmt.Printf("%v all stream finished\n", time.Now().Second()*1000+time.Now().Nanosecond()/1000000.0)

	//fmt.Printf("total time: %v", time.Since(t0).Nanoseconds()/1000000.0)

	session.Close(nil)

}

// Start a server that echos all data on the first stream opened by the client
func serverMain(c *config) {
	listener, err := quic.ListenAddr("0.0.0.0:"+strconv.Itoa(c.Port), generateTLSConfig(c), nil)
	quic_utils.Check(err)

	sess, err := listener.Accept()
	quic_utils.Check(err)

	for {
		stream, err := sess.AcceptStream()
		quic_utils.Check(err)

		go func() {
			sizeBuf := make([]byte, 64, 64)
			nrecv, err := stream.Read(sizeBuf)
			quic_utils.Check(err)
			intSize, err := strconv.ParseInt(string(sizeBuf[:nrecv]), 10, 0)
			quic_utils.Check(err)

			sendBuf := make([]byte, intSize, intSize)
			n := 0
			for n < int(intSize) {
				add, err := stream.Write(sendBuf)
				quic_utils.Check(err)
				n += add
			}

			waitBuf := make([]byte, 0, 0)
			for {
				_, err = stream.Read(waitBuf)
				quic_utils.Check(err)
			}
		}()
	}
}

// Setup a bare-bones TLS config for the server
func generateTLSConfig(conf *config) *tls.Config {

	pub, err := quic_utils.ExtractPublicKey(conf.Public)
	quic_utils.Check(err)
	priv, err := quic_utils.ExtractPrivateKey(conf.Private)
	quic_utils.Check(err)
	cert, err := quic_utils.MakeCertificate(pub, priv)
	quic_utils.Check(err)

	return &tls.Config{Certificates: []tls.Certificate{cert}}
}
