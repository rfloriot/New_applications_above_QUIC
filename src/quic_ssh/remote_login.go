package main

import (
	"github.com/lucas-clemente/quic-go"
	"io"
	"strings"
	"os/exec"
	"fmt"
	"os"
	"time"
	"os/signal"
)

/////////////////
// server part //
/////////////////

func remoteLoginServerLoops(stream quic.Stream, serverConfig *SSHServer, stopChanel chan bool) {
	errorChannel := make(chan error)
	inputReader, inputWriter := io.Pipe()
	outputReader, outputWriter := io.Pipe()
	errorReader, errorWriter := io.Pipe()

	commands := make(chan []byte, serverConfig.conf.bufSize)

	go receiveCommand(errorChannel, serverConfig, inputWriter, stream, commands)
	go runTelnet(errorChannel, serverConfig, inputReader, outputWriter, errorWriter)
	go sendOutputResult(errorChannel, serverConfig, outputReader, stream)
	go lookForEndOfCommunication(errorChannel, serverConfig, errorReader)

	// wait for end of service
	<-errorChannel
	stopChanel <- true
}

// receives the commands and decide whether to send it directly to telnet or to do something locally
func receiveCommand(communicationChannel chan error, serverConf *SSHServer, in writable, stream quic.Stream, commands chan []byte) {
	for {
		var msg []byte
		readBuffer := make([]byte, serverConf.conf.bufSize, serverConf.conf.bufSize)
		n, err := stream.Read(readBuffer)
		msg = readBuffer[:n]
		if n > 0 {
			//this message should be transmitted to telnet
			in.Write(msg)
			if serverConf.conf.testMode {
				serverConf.conf.testOutput = serverConf.conf.testOutput + "\n" + string(msg)
			}
		}
		if (err != nil) {
			communicationChannel <- err
		}
	}
}

func runTelnet(communicationChannel chan error, serverConf *SSHServer, in readable, out writable, out_err writable) {
	cmd := exec.Command("telnet", "localhost", "5051")
	cmd.Stdout = out
	cmd.Stderr = out_err
	cmd.Stdin = in
	err := cmd.Run()
	if err != nil {
		out.Write([]byte(fmt.Sprintf("Error : %s\n", err)))
	}
}

func runTelnetd() {
	cmd := exec.Command("busybox", "telnetd", "-F", "-p", "5051")
	cmd.Start()
	go func() {
		sigchan := make(chan os.Signal, 10)
		signal.Notify(sigchan, os.Interrupt)
		<-sigchan
		os.Exit(0)
	}()
	cmd.Wait()

}

func sendOutputResult(communicationChannel chan error, serverConf *SSHServer, in readable, stream quic.Stream) {
	readBuffer := make([]byte, serverConf.conf.bufSize, serverConf.conf.bufSize)
	for {
		n, err := in.Read(readBuffer)
		if err == nil {
			msg := readBuffer[:n]
			stream.Write(msg)

		} else {
			communicationChannel <- err
			return
		}
	}
}

// look on stderr of telnet to find end of connection
func lookForEndOfCommunication(communicationChannel chan error, serverConf *SSHServer, out_err readable) {
	readBuffer := make([]byte, serverConf.conf.bufSize, serverConf.conf.bufSize)
	for {
		n, err := out_err.Read(readBuffer)
		msg := string(readBuffer[:n])
		if err == nil {
			if strings.Contains(msg, "Connection closed by foreign host") {
				communicationChannel <- io.EOF
			}
		}
	}
}

/////////////////
// client part //
/////////////////

func disableEcho() {
	cmd := exec.Command("stty", "-echo", "-icanon", "min", "1")
	cmd.Stdin = os.Stdin
	cmd.Output()
}

func reEnableEcho() {
	cmd := exec.Command("stty", "echo", "icanon")
	cmd.Stdin = os.Stdin
	cmd.Output()
}

func remoteLoginClientLoops(stream quic.Stream, clientConfig *SSHClient, stopChanel chan bool) {
	errorChannel := make(chan error)
	disableEcho()

	go writeMessageLoop(errorChannel, clientConfig, os.Stdin, stream)
	go receiveMessageLoop(errorChannel, clientConfig, stream, os.Stdout)

	go func() {
		sigchan := make(chan os.Signal, 10)
		signal.Notify(sigchan, os.Interrupt)
		for true{
			<-sigchan
			var ctrlC []byte = nil
			ctrlC = append(ctrlC, byte(0x03))
			stream.Write(ctrlC)
		}
	}()

	// wait for end of service
	<-errorChannel
	reEnableEcho()
	stopChanel <- true
}

// stdin -> prepare for sending
func writeMessageLoop(communicationChannel chan error, c *SSHClient, in readable, stream quic.Stream) {
	readBuffer := make([]byte, c.conf.bufSize, c.conf.bufSize)
	for {
		if c.conf.testInput != "" { // automatic test case. Content of this if case used only when launched in 'go test'
			msg := strings.Split(c.conf.testInput, "\n")[0] + "\n"
			stream.Write([]byte(msg))
			time.Sleep(200 * time.Millisecond)
		} else {
			n, err := in.Read(readBuffer)
			if err == nil {
				msg := readBuffer[:n]
				stream.Write(msg)
			} else {
				communicationChannel <- err
				return
			}
		}
	}
}

// message received on stream -> stdout.
func receiveMessageLoop(communicationChannel chan error, c *SSHClient, stream quic.Stream, out writable) {
	for {
		readBuffer := make([]byte, c.conf.bufSize, c.conf.bufSize)
		n, err := stream.Read(readBuffer)
		if n > 0 {
			msg := readBuffer[:n]
			if c.conf.testMode { // only for test mode, stop after telnet first handshake
				c.conf.testOutput = string(msg)
				stream.Close()
				c.session.Close(nil)
			}else{
				//show msg to the user
				out.Write(msg)
			}

		}
		if err != nil {
			communicationChannel <- err
			return
		}
	}
}
