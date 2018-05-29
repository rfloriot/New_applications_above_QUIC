package main

import (
	"testing"
	"strings"
	"os/exec"
)

func init() {
	logTmp("5")
}

func launchServerWithResult(port int, conf *SSHConfig) {
	conf.bufSize = 100000
	conf.port = port
	conf.testMode = true
	conf.privKeyFile = directory + "pr_server"
	conf.pubKeyFile = directory + "pk_server"
	sshServer := NewQuicSSHServer(conf)
	sshServer.Run()
}

func TestRemoteLogin(t *testing.T) {
	port := 41112
	confServer := SSHConfig{}
	go launchServerWithResult(port, &confServer)
	writeFile(directory+"known_hosts_client", "127.0.0.1:41112 "+dummyServerPublicKeyInline)
	conf := SSHConfig{}
	conf.bufSize = 100000
	conf.testMode = true
	conf.hostname = "127.0.0.1"
	conf.port = port
	conf.privKeyFile = directory + "pr_client"
	conf.pubKeyFile = directory + "pk_client"
	conf.testInput = "test\n"
	sshClient := NewQuicSSHClient(&conf)
	sshClient.Run()
	if !strings.Contains(confServer.testOutput, "test") {
		t.Errorf("Error with remote login: cannot submit command to server")
	}
	if !strings.Contains(conf.testOutput, "Trying 127.0.0.1...") && ! strings.Contains(conf.testOutput, "Trying ::1...") {
		t.Errorf("Error with remote login: cannot receive answer from server, received : %s", conf.testOutput)
	}

	// test a second client with also port forwarding active:
	conf = SSHConfig{}
	conf.bufSize = 100000
	conf.testMode = true
	conf.hostname = "127.0.0.1"
	conf.port = port
	conf.localPort = 9876
	conf.remotePort = 5432
	conf.localPortForwarding = true
	_, conf.remoteIP = resolveHostname("127.0.0.1")
	conf.privKeyFile = directory + "pr_client"
	conf.pubKeyFile = directory + "pk_client"
	conf.testInput = "test\n"
	sshClient = NewQuicSSHClient(&conf)
	sshClient.Run()
	if !strings.Contains(confServer.testOutput, "test") {
		t.Errorf("Error with remote login: cannot submit command to server")
	}
	if !strings.Contains(conf.testOutput, "Trying 127.0.0.1...") && ! strings.Contains(conf.testOutput, "Trying ::1...") {
		t.Errorf("Error with remote login: cannot receive answer from server")
	}

	cmd := exec.Command("pkill", "busybox")
	cmd.Start()
	cmd.Wait()
}
