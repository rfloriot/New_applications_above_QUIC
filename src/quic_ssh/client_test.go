package main

import (
	"quic_utils"
	"testing"
	"strings"
	"io/ioutil"
	"crypto/rsa"
	"bytes"
)

func init() {
	logTmp("3")
}

func TestAskForUnknownRemotePublicKey(t *testing.T) {
	pk, err := quic_utils.ExtractPublicKey(directory + "pk_server")
	conf := SSHConfig{}
	conf.testMode = true
	conf.authorizedPublicKeysFile = directory + "known_hosts_client_void"
	if err != nil {
		t.Errorf("cannot extract a public key from file. Stop the test")
	}

	// test1: accept server key
	si := serverInfo{"111.222.3.4", pk}
	conf.testInput = "y"
	result := askForUnknownRemotePublicKey(si, &conf)
	if !result {
		t.Errorf("should return true. False received")
	}
	dat, err := ioutil.ReadFile(directory + "known_hosts_client_void")
	if err != nil {
		t.Errorf("Cannot read file that was modified when adding new server public key.")
	}
	parts := strings.Split(string(dat), "\n")
	content := ""
	for _, elem := range parts {
		if len(elem) > 0 {
			if content != "" {
				t.Errorf("There should not be more than 1 key (1 line of data) in this known_hosts file")
			} else {
				content = elem
			}
		}
	}
	if content != "111.222.3.4 "+dummyServerPublicKeyInline {
		t.Errorf("bad content for added line in known_hosts file")
	}


	// test2: refuse server key
	writeFile(directory+"known_hosts_client_void", "")
	conf.testInput = "n"
	result = askForUnknownRemotePublicKey(si, &conf)
	if result {
		t.Errorf("should return false but true was received")
	}

	// test3: write something crazy in place of 'y' or 'n'
	conf.testInput = "something wrong"
	result = askForUnknownRemotePublicKey(si, &conf)
	dat, err = ioutil.ReadFile(directory + "known_hosts_client_void")
	if err != nil {
		t.Errorf("Cannot read file that was modified when adding new server public key.")
	}
	parts = strings.Split(string(dat), "\n")
	for _, elem := range parts {
		if len(elem) > 0 {
			t.Errorf("Known_host file should not contains additional key if client said 'n' or something wrong")
		}
	}
}

func comparePrivateKeys(pr1 *rsa.PrivateKey, pr2 *rsa.PrivateKey, t *testing.T) bool {
	if bytes.Compare(pr1.D.Bytes(), pr2.D.Bytes()) != 0 {
		return false
	}
	if len(pr1.Primes) != len(pr2.Primes) {
		return false
	}
	for i := 0; i < len(pr1.Primes); i++ {
		if bytes.Compare(pr1.Primes[i].Bytes(), pr2.Primes[i].Bytes()) != 0 {
			return false
		}
	}
	return true
}

func TestInitServerAndClient(t *testing.T) {
	go launchServer(41111)
	port := 41111
	conf := SSHConfig{}
	conf.bufSize = 100000
	conf.testInput = "n"
	conf.hostname = "127.0.0.1"
	conf.authorizedPublicKeysFile = directory + "known_hosts_client_with_invalid_key"
	conf.port = port
	conf.testMode = true
	conf.privKeyFile = directory + "pr_client"
	conf.pubKeyFile = directory + "pk_client"
	sshClient := NewQuicSSHClient(&conf)

	if sshClient != nil{
		t.Errorf("SSHClient should be nil when user rejects server's key")
	}
}
