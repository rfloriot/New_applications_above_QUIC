package main

import (
	"quic_utils"
	"github.com/lucas-clemente/quic-go"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"bytes"
	"strings"
	"encoding/pem"
	"os"
	"errors"
	"os/signal"
)

type SSHClient struct {
	conf        *SSHConfig
	session     quic.Session
	firstStream quic.Stream
	publicKey   *rsa.PublicKey
	privateKey  *rsa.PrivateKey
	stopChannel chan bool
}

func NewQuicSSHClient(config *SSHConfig) (*SSHClient) {
	// Step 1) contacting distant server to open session and opening the first stream
	session, err := config.openSession()
	quic_utils.Check(err)
	cert := config.getServerCert(session)
	stream, err := session.OpenStreamSync()
	quic_utils.Check(err)

	// Step 2) authenticate and then allow or reject this server given it's public key
	if !config.allowServer(session, cert.PublicKey.(*rsa.PublicKey)) {
		session.Close(nil)
		return nil
	}

	// Step 3) extracting our public and private keys from files
	publicKey, err := quic_utils.ExtractPublicKey(config.pubKeyFile)
	quic_utils.Check(err)
	privateKey, err := quic_utils.ExtractPrivateKey(config.privKeyFile)
	quic_utils.Check(err)

	// return an object regrouping all variables needed for running the client
	return &SSHClient{
		conf:        config,
		session:     session,
		firstStream: stream,
		publicKey:   publicKey,
		privateKey:  privateKey,
		stopChannel: make(chan bool),
	}
}

// Run the program in client mode. This is done when stream is already opened by creating SSHClient instance
func (c *SSHClient) Run() error {

	// Step 4) give our public key (application level) + sign with our private key
	quic_utils.ServeClientPublicKey(c.session, c.firstStream, c.privateKey, c.publicKey)

	// Step 5) tell to server the mode to use (port forwarding and/or remote login)
	c.setServerMode()

	// Step 6) [optional] launch local port forwarding
	if c.conf.localPortForwarding {
		c.launchPortForwarding(true)
	}

	// Step 7) [optional] launch remote port forwarding
	if c.conf.remotePortForwarding && !c.conf.localPortForwarding {
		c.launchPortForwarding(false)
	}

	// Step 8) [optional] launch remote login
	if !c.conf.onlyForwardPort {
		c.launchRemoteLogin()
	}

	// Step 9) [optional] wait for "exit" msg from user to stop port forwarding
	if c.conf.onlyForwardPort {
		c.waitForExitRequest()
	}

	// Step 10) wait for message received on stopChannel then close the session
	// such stop message can come from step 6 or 7 from inside goroutines.
	<-c.stopChannel
	c.session.Close(nil)
	return nil
}

func (conf *SSHConfig) allowServer(session quic.Session, serverPK *rsa.PublicKey) (result bool) {
	if conf.authorizedPublicKeysFile == "" {
		return true; // if it was not requested to verify server's public key
	}
	resultCheck, remoteServer := checkRemotePublicKey(conf, session.RemoteAddr().String(), serverPK)
	if !resultCheck {
		if !askForUnknownRemotePublicKey(remoteServer, conf) { // ask client if he trusts the server
			return false
		}
	}
	return true
}

// verify if received public key is in our known hosts file, else ask if we trust this connection
func checkRemotePublicKey(conf *SSHConfig, remoteAddr string, serverPk *rsa.PublicKey) (bool, serverInfo) {
	knownServers := getKnownHosts(conf.authorizedPublicKeysFile)
	remoteServer := getRemoteServerInfos(remoteAddr, serverPk)
	i := 0;
	found := false
	for i < len(knownServers) && !found {
		if knownServers[i].ip == remoteServer.ip && quic_utils.ComparePublicKeys(knownServers[i].publicKey, remoteServer.publicKey) {
			found = true
		}
		i = i + 1
	}
	if !found {
		return false, remoteServer
	}
	return true, remoteServer
}

// ask to user if the remote host can be trusted. If trusted, add [IP,Key] to known hosts file
func askForUnknownRemotePublicKey(remoteServer serverInfo, conf *SSHConfig) bool {
	keyPem, err := quic_utils.EncodePublicKey(remoteServer.publicKey)
	quic_utils.Check(err)
	bloc := pem.Block{"RSA PUBLIC KEY", nil, keyPem}
	keyHex := pem.EncodeToMemory(&bloc)
	conf.printMsg(fmt.Sprintf("\nThe host %s is not known on this computer. Its public key is:\n%s\n\nDo you stil want to connect to this host (yes/no)?", remoteServer.ip, removePemMarkers(keyHex)))
	yes := []string{"yes", "YES", "y", "Y"}
	no := []string{"no", "NO", "n", "N"}
	answer := ""
	result := ""
	if conf.testMode {
		answer = conf.testInput
	} else {
		fmt.Scanln(&answer)
	}
	if answer == yes[0] || answer == yes[1] || answer == yes[2] || answer == yes[3] {
		result = "yes"
	} else if answer == no[0] || answer == no[1] || answer == no[2] || answer == no[3] {
		result = "no"
	} else {
		result = ""
	}
	if result != "yes" {
		return false
	}
	f, err := os.OpenFile(conf.authorizedPublicKeysFile, os.O_APPEND|os.O_WRONLY, 0600)
	quic_utils.Check(err)
	line := "\n" + remoteServer.ip + " " + removePemMarkers(keyHex)
	f.Write([]byte(line))
	err = f.Sync()
	quic_utils.Check(err)
	err = f.Close()
	quic_utils.Check(err)
	return true
}

// open known hosts file to produce a list of trusted (ip, publicKey)
func getKnownHosts(file string) []serverInfo {
	var result []serverInfo
	data, err := ioutil.ReadFile(file)
	quic_utils.Check(err)
	parts := bytes.Split(data, []byte("\n"))
	for _, line := range parts {
		if len(line) > 2 && !strings.HasPrefix(string(line), "--") {
			subparts := bytes.Split(line, []byte(" "))
			newHost := serverInfo{"", nil}
			pemDataWasValid := true
			for _, bloc := range subparts {
				if len(bloc) > 0 {
					if newHost.ip == "" {
						newHost.ip = string(bloc)
						quic_utils.Check(err)
					} else {
						key := addPemMarkers(bloc)
						pemData, _ := pem.Decode(key)
						if pemData != nil {
							asnData := pemData.Bytes
							newHost.publicKey, err = quic_utils.DecodePublicKey(asnData)
							quic_utils.Check(err)
						} else {
							pemDataWasValid = false
						}
					}
				}
			}
			if pemDataWasValid && newHost.publicKey != nil {
				result = append(result, newHost)
			}
		}
	}
	return result
}

// return a serverInfo (i.e. a tuple [ip, public key]) for current remote server
func getRemoteServerInfos(remoteAddr string, serverPk *rsa.PublicKey) serverInfo {
	result := serverInfo{}
	result.ip = remoteAddr
	result.publicKey = serverPk
	return result
}

/*
 * after getting allowed, first message of the client sent is:
 * > "1" if the client wants remote login only
 * > "2" if the client wants port forwarding only
 * > "3" if the client wants both remote login and port forwarding
 * This method write on the stream this number
 */
func (c *SSHClient) setServerMode() (error) {
	var n int
	var err error
	if c.conf.onlyForwardPort {
		n, err = c.firstStream.Write([]byte("2"))
	} else if c.conf.localPortForwarding || c.conf.remotePortForwarding {
		n, err = c.firstStream.Write([]byte("3"))
	} else {
		n, err = c.firstStream.Write([]byte("1"))
	}
	if err != nil || n != 1 {
		return errors.New("a problem appeared when writing server mode on stream")
	}
	return nil
}

func (c *SSHClient) launchPortForwarding(local bool) {
	initialForwardingConfig := newPortForwardingSession(c.conf, c.session, c.firstStream)
	if (local) {
		go initialForwardingConfig.runAsSource(c.conf.localPort, c.conf.remotePort, c.conf.remoteIP)
	} else {
		go initialForwardingConfig.runAsDestination()
		stream, err := c.session.OpenStreamSync()
		quic_utils.Check(err)
		err = writeControlMessage(stream, false, uint16(c.conf.localPort), uint16(c.conf.remotePort), c.conf.remoteIP)
		if err != nil {
			c.conf.printMsg("A problem appeared when trying to established port forwarding. Stopping port forwarding")
			c.session.Close(nil)
			os.Exit(-1)
		}
	}
}

func (c *SSHClient) launchRemoteLogin() {
	go remoteLoginClientLoops(c.firstStream, c, c.stopChannel)
}

// this method simply waits that user enter "exit" on command line when port forwarding is active to stop it.
// It also reads the stream to show eventual error message coming from server.
func (c *SSHClient) waitForExitRequest() {
	c.conf.printMsg("Port forwarding active. Press Ctrl-C to stop it.")
	go func() {
		for {
			readBuffer := make([]byte, c.conf.bufSize, c.conf.bufSize)
			n, err := c.firstStream.Read(readBuffer)
			if n > 0 {
				c.conf.printMsg(fmt.Sprintf("%s", readBuffer[:n]))
			}
			if err != nil {
				c.stopChannel <- true
				return
			}
		}
	}()
	go func() {
		if !c.conf.testMode { // stop when Ctrl-C
			sigchan := make(chan os.Signal, 10)
			signal.Notify(sigchan, os.Interrupt)
			<-sigchan
			c.conf.printMsg("\n	Bye")
			c.stopChannel <- true
			return
		} else { // automatic test case, stop when something arrive on c.conf.testStopClient
			readBuffer := make([]byte, c.conf.bufSize, c.conf.bufSize)
			c.conf.testStopClient.Read(readBuffer)
			c.stopChannel <- true
		}
	}()
}
