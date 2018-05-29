package main

import (
	"os"
	"fmt"
	"strconv"
	"strings"
)

//parse all the command line arguments
func (conf *SSHConfig) parseArguments() {
	unparsed := make([]string, 0)
	conf.bufSize = 100000

	for index := 1; index < len(os.Args); index++ {
		unparsed, index = parseOneArgument(conf, index, unparsed)
	}

	if conf.onlyForwardPort && !conf.remotePortForwarding && !conf.localPortForwarding{
		usage("Argument -N cannot be used if no port forwarding is requested", conf)
	}

	if len(unparsed) == 2 {
		conf.hostname = unparsed[0]
	}
	if len(unparsed) >= 1 {
		port, err := strconv.Atoi(unparsed[len(unparsed)-1])
		if err != nil {
			usage("invalid port value'" + unparsed[len(unparsed)-1] + "' ", conf)
		} else {
			conf.port = port
		}
	}
	if len(unparsed) > 2{
		usage("invalid command'" + unparsed[len(unparsed)-1] + "' ", conf)
	}

	if !conf.listen {
		if conf.hostname == "" || conf.port == 0 {
			usage("In non listen mode you should define an address to connect", conf)
		}
	}

	if conf.listen && (conf.privKeyFile == "" || conf.pubKeyFile == "") {
		usage("must specify keys for server", conf)
	}
}

//handle one command line argument, update config and add not parsed arguments to 'unparsed'
func parseOneArgument(conf *SSHConfig, i int, unparsed []string) ([]string, int) {
	var err error
	switch os.Args[i] {
	case "-b":
		conf.bufSize, err = strconv.Atoi(os.Args[i+1])
		i++
		if err != nil{
			usage("Buffer size not correct. Should be integer.", conf)
		}
	case "-h":
		usage("", conf)
	case "-l":
		conf.listen = true
	case "-L":
		if conf.remotePortForwarding{
			usage("Cannot create remote and local port forwarding from a single call", conf)
		}
		conf.localPortForwarding = true
		str := strings.Split(os.Args[i+1], ":")
		if len(str) < 3{
			usage("Bad argument for local port forwarding", conf)
		}
		val, err := strconv.Atoi(str[0])
		conf.localPort = uint16(val)
		if err != nil {
			usage("Local port not correct. Should be integer.", conf)
		}
		val , err = strconv.Atoi(str[len(str)-1])
		conf.remotePort = uint16(val)
		if err != nil {
			usage("Remote port not correct. Should be integer.", conf)
		}
		hostnameStr := os.Args[i+1][len(str[0])+1:len(os.Args[i+1])-(len(str[len(str)-1])+1)]
		if strings.Index(hostnameStr, "[") == 0 &&
			strings.LastIndex(hostnameStr, "]") == len(hostnameStr)-1 {
			hostnameStr = hostnameStr [1:len(hostnameStr)-1]
		}
		err, remoteIP := resolveHostname(hostnameStr)
		if err != nil{
			usage("Remote hostname cannot be resolved.", conf)
		}
		conf.remoteIP = remoteIP
		i++
	case "-N":
		conf.onlyForwardPort = true
	case "-R":
		if conf.localPortForwarding{
			usage("Cannot create remote and local port forwarding from a single call", conf)
		}
		conf.remotePortForwarding = true
		str := strings.Split(os.Args[i+1], ":")
		if len(str) != 3{
			usage("Bad argument for remote port forwarding", conf)
		}
		val, err := strconv.Atoi(str[0])
		conf.localPort = uint16(val)
		if err != nil {
			usage("Local port not correct. Should be integer.", conf)
		}
		val , err = strconv.Atoi(str[len(str)-1])
		conf.remotePort = uint16(val)
		if err != nil {
			usage("Remote port not correct. Should be integer.", conf)
		}
		hostnameStr := os.Args[i+1][len(str[0])+1:len(os.Args[i+1])-(len(str[len(str)-1])+1)]
		if strings.Index(hostnameStr, "[") == 0 &&
			strings.LastIndex(hostnameStr, "]") == len(hostnameStr)-1 {
			hostnameStr = hostnameStr [1:len(hostnameStr)-1]
		}
		err, remoteIP := resolveHostname(hostnameStr)
		if err != nil{
			usage("Remote hostname cannot be resolved.", conf)
		}
		conf.remoteIP = remoteIP
		i++
	case "--priv":
		conf.privKeyFile = os.Args[i+1]
		i++
	case "--pub":
		conf.pubKeyFile = os.Args[i+1]
		i++
	case "--req":
		conf.authorizedPublicKeysFile = os.Args[i+1]
		i++
	case "--user":
		conf.username = os.Args[i+1]
		i++
	case "--pass":
		conf.password = os.Args[i+1]
		i++
	default:
		unparsed = append(unparsed, os.Args[i])
	}
	return unparsed, i
}

func usage(message string, conf *SSHConfig) {
	buf := ""
	if message != "" {
		buf += "[Error] " + message + "\n\n"
	}
	buf += "QuicSSH\n"
	buf += "Usage: quic_ssh [options] [hostname] [port]\n"
	buf += "\n"
	buf += "-b       internal buffer size (default=100000)\n"
	buf += "-l       Bind and listen for incoming connections\n"
	buf += "-L       makes port forwarding by using syntax: localPort:hostname:remotePort\n"
	buf += "-N       only forward ports, do not open interactive ssh session"
	buf += "-R       makes port forwarding by using syntax: remotePort:hostname:localPort\n"
	buf += "--priv   Private key location (required if -l set)\n"
	buf += "--pub    Public key location (required if -l set)\n"
	buf += "--req    authorized_keys file for server / known_hosts file for client (if check required)\n"
	buf += "\nOther options on the client for measurements/debugging only:\n"
	buf += "--pass   set the password directly in the arguments\n"
	buf += "--user   set the username directly in the arguments\n"
	buf += ""

	if !conf.testMode{
		fmt.Println(buf)
		os.Exit(1)
	}else{
		conf.testOutput=buf
	}
}