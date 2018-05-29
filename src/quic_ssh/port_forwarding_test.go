package main

import (
	"testing"
	"os/exec"
	"time"
	"io"
	"io/ioutil"
	"bytes"
	"strings"
	"fmt"
)

func init() {
	logTmp("4")
}

func launchPortForwardingClient(port int, local bool, localPort int, remotePort int, stopClient readable) {
	conf := SSHConfig{}
	conf.bufSize = 100000
	conf.testMode = true
	conf.hostname = "127.0.0.1"
	writeFile(directory+"known_hosts_client", fmt.Sprintf("127.0.0.1:%d ", port)+dummyServerPublicKeyInline)
	conf.authorizedPublicKeysFile = directory + "known_hosts_client"
	conf.port = port
	conf.privKeyFile = directory + "pr_client"
	conf.pubKeyFile = directory + "pk_client"
	conf.onlyForwardPort = true
	conf.testStopClient = stopClient
	if local {
		conf.localPortForwarding = true
	} else {
		conf.remotePortForwarding = true
	}
	_, conf.remoteIP = resolveHostname("127.0.0.1")
	conf.localPort = uint16(localPort)
	conf.remotePort = uint16(remotePort)
	sshClient := NewQuicSSHClient(&conf)
	sshClient.Run()
}

func launchServer(port int) {
	conf := SSHConfig{}
	conf.bufSize = 100000
	conf.port = port
	conf.listen = true
	conf.testMode = true
	conf.privKeyFile = directory + "pr_server"
	conf.authorizedPublicKeysFile = directory + "authorized_hosts_server"
	conf.pubKeyFile = directory + "pk_server"
	sshServer := NewQuicSSHServer(&conf)
	sshServer.Run()
}

/*
 * launch command
 * nc -l -p portToListen > directory/nc_dst_testID
 */
func launchNetcatServer(testID string, portToListen string, t *testing.T) {
	blockingStdin, _ := io.Pipe()
	outputReader, outputWriter := io.Pipe()
	var stderr bytes.Buffer
	go func() {
		result := ""
		readBuffer := make([]byte, 1000000, 1000000)
		for {
			n, err := outputReader.Read(readBuffer)
			if err == nil {
				result = result + string(readBuffer[:n])
				writeFile(directory+"nc_dst_"+testID, result)
			} else {
				t.Errorf("Error when using a netcat server : %s\n", err)
			}
		}
	}()
	cmd := exec.Command("nc", "-l", "-p", portToListen)
	//var stdout, stderr bytes.Buffer
	cmd.Stdout = outputWriter
	cmd.Stderr = &stderr
	cmd.Stdin = blockingStdin
	err := cmd.Run()
	if err != nil {
		t.Errorf("Cannot launch netcat server. Test failed to launch. %s\n:%s\n", err, string(stderr.Bytes()))
	}

}

/*
 * launch command
 * netcat -l 127.0.0.1 -p portToListen < directory/nc_src_testID
 */
func launchNetcatClient(testID string, portToContact string, t *testing.T) {
	inputReader, inputWriter := io.Pipe()
	dat, err := ioutil.ReadFile(directory + "nc_src_" + testID)
	if err != nil {
		t.Errorf("Cannot launch netcat client. Test failed when reading on /tmp %s\n", err)
	}
	go func() {
		inputWriter.Write(dat)
		inputWriter.Close()
	}()

	cmd := exec.Command("nc", "127.0.0.1", portToContact)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Stdin = inputReader
	err = cmd.Run()
	if err != nil {
		t.Errorf("Cannot launch netcat client. Test failed to launch. %s : %s\n", err, string(stderr.Bytes()))
	}
}

func createLargeFile(sizeInBytes string, t *testing.T, testID string) {
	cmd := exec.Command("head", "-c", sizeInBytes, "/dev/urandom")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		t.Errorf("Problem when running tests when creating file from /dev/urandom %s\n", string(stderr.Bytes()))
	}
	outStr, _ := string(stdout.Bytes()), string(stderr.Bytes())
	writeFile(directory+"nc_src_"+testID, outStr)
}

func getSha256OfFile(file string) string {
	cmd := exec.Command("sha256sum", file)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err != nil {
		return ""
	}
	outStr, _ := string(stdout.Bytes()), string(stderr.Bytes())
	return strings.Split(outStr, " ")[0]
}

func compareHash(testID string)(bool){
	hash1 := getSha256OfFile(directory+"nc_src_"+testID)
	hash2 := getSha256OfFile(directory+"nc_dst_"+testID)
	return hash1 != "" && hash2 != "" && hash1 == hash2
}

func TestLocalPortForwarding(t *testing.T) {
	stopClientReader, stopClientWriter := io.Pipe()
	port := 41113
	localPort := 42222
	remotePort := 43333
	go launchServer(port)
	go launchPortForwardingClient(port, true, localPort, remotePort, stopClientReader)
	time.Sleep(500 * time.Millisecond)

	// verify that files are correctly transfered using netcat on top of local port forwarding

	// very short file
	writeFile(directory+"nc_src_1", "Short content!")
	go launchNetcatServer("1", "43333", t)
	time.Sleep(50 * time.Millisecond)
	launchNetcatClient("1", "42222", t)
	for i := 0; i < 30; i++{
		time.Sleep(50 * time.Millisecond)
		if compareHash("1") {
			break
		}
	}
	if !compareHash("1") {
		t.Errorf("Cannot transfer properly short file with netcat on top of local port forwarding\n")
	}

	// 1Mo file
	createLargeFile("1000000", t, "2")
	go launchNetcatServer("2", "43333", t)
	time.Sleep(50 * time.Millisecond)
	launchNetcatClient("2", "42222", t)
	for i := 0; i < 30; i++{
		time.Sleep(50 * time.Millisecond)
		if compareHash("2") {
			break
		}
	}
	stopClientWriter.Write([]byte("stop"))
	time.Sleep(100 * time.Millisecond)
	if !compareHash("2") {
		t.Errorf("Cannot transfer properly 1Mo file with netcat on top of local port forwarding\n")
	}
}

func TestRemotePortForwarding(t *testing.T) {
	stopClientReader, stopClientWriter := io.Pipe()
	port := 41114
	localPort := 42224
	remotePort := 43335
	go launchServer(port)
	go launchPortForwardingClient(port, false, localPort, remotePort, stopClientReader)
	time.Sleep(500 * time.Millisecond)

	// verify that files are correctly transfered using netcat on top of local port forwarding
	//// very short file
	writeFile(directory+"nc_src_3", "Short content!")
	go launchNetcatServer("3", "43335", t)
	time.Sleep(100 * time.Millisecond)
	launchNetcatClient("3", "42224", t)
	for i := 0; i < 30; i++{
		logTmp("ici3")
		time.Sleep(50 * time.Millisecond)
		if compareHash("3") {
			break
		}
	}
	if !compareHash("3") {
		t.Errorf("Cannot transfer properly short file with netcat on top of remote port forwarding\n")
	}

	//// 1Mo file
	createLargeFile("1000000", t, "4")
	go launchNetcatServer("4", "43335", t)
	time.Sleep(100 * time.Millisecond)
	launchNetcatClient("4", "42224", t)
	for i := 0; i < 30; i++{
		logTmp("ici4")
		time.Sleep(50 * time.Millisecond)
		if compareHash("4") {
			break
		}
	}
	stopClientWriter.Write([]byte("stop"))
	time.Sleep(100 * time.Millisecond)
	if !compareHash("4") {
		t.Errorf("Cannot transfer properly 1Mo file with netcat on top of remote port forwarding\n")
	}

}
