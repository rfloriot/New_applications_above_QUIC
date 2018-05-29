package main

import (
	"quic_utils"
)

func main() {
	conf := SSHConfig{}
	conf.parseArguments()

	if conf.listen {
		sshServer := NewQuicSSHServer(&conf)
		quic_utils.Check(sshServer.Run())
	} else {
		sshClient := NewQuicSSHClient(&conf)
		if sshClient != nil{
			quic_utils.Check(sshClient.Run())
		}

	}

}
