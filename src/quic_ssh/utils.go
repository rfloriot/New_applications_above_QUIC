package main

import (
	"crypto/rsa"
	"fmt"
	"net"
	"strconv"
	"errors"
	"encoding/binary"
)

type SSHConfig struct {
	bufSize                  int
	pubKeyFile               string
	privKeyFile              string
	publicKey                *rsa.PublicKey
	privateKey               *rsa.PrivateKey
	authorizedPublicKeysFile string // file with multiple allowed remote public keys
	username                 string //username and password can be set directly in arguments (useful to make time measurements)
	password                 string
	listen                   bool   // is server ?
	hostname                 string // if client, hostname to contact
	port                     int    // if client, port to contact
	localPortForwarding      bool   // if client, launched with -L ?
	remotePortForwarding     bool   // if client, launched with -R ?
	onlyForwardPort          bool   // if client, launched with -N ?

	//if local/remote port forwarding used:
	localPort  uint16
	remotePort uint16
	remoteIP   net.IP

	// additional variables for automatic unit tests:
	testMode       bool
	testInput      string
	testOutput     string
	testStopClient readable
}

const PRINT_LEVEL_DEBUG = 1
const PRINT_LEVEL_NORMAL = 0
const print_level = PRINT_LEVEL_DEBUG

func (c *SSHConfig) printDebug(str string){
	if !c.testMode && print_level == PRINT_LEVEL_DEBUG {
		fmt.Printf("[Debug] %s\n", str)
	}
}
func (c *SSHConfig) printMsg(str string){
	if !c.testMode {
		fmt.Printf("%s\n", str)
	}
}

func (c *SSHConfig) formatAddress() string {
	return fmt.Sprintf("%v:%v", c.hostname, c.port)
}

type closable interface {
	Close() error
}

type serverInfo struct {
	ip        string
	publicKey *rsa.PublicKey
}

type readable interface {
	Read(b []byte) (n int, err error)
}

type writable interface {
	Write(b []byte) (n int, err error)
}

/*
 * this function returns only 1 ip (even if multiple ips were found).
 */
func resolveHostname(hostname string) (err error, ip []byte) {
	ips, err := net.LookupIP(hostname)
	if err != nil || len(ips) == 0 {
		return errors.New("error when trying to resolve hostname"), nil
	}

	var firstV4 net.IP = nil
	var firstV6 net.IP = nil
	i := 0
	for (firstV4 == nil || firstV6 == nil) && i < len(ips) {
		if net.IP.To4(ips[i]) != nil && firstV4 == nil {
			firstV4 = net.IP.To4(ips[i])
		} else if net.IP.To16(ips[i]) != nil && firstV6 == nil {
			firstV6 = net.IP.To16(ips[i])
		}
		i++
	}

	if firstV6 != nil {
		return nil, firstV6
	}
	return nil, firstV4
}

/*
 * convert []byte ip into string for using net.Dial
 * empty string is returned in case of error
 */
func ipToString(ip []byte) string {
	if len(ip) != 4 && len(ip) != 16 {
		return ""
	}
	if len(ip) == 4 {
		return strconv.Itoa(int(ip[0])) + "." + strconv.Itoa(int(ip[1])) + "." +
			strconv.Itoa(int(ip[2])) + "." + strconv.Itoa(int(ip[3]))
	}
	result := fmt.Sprintf("%02x", int(ip[0])) + fmt.Sprintf("%02x", int(ip[1]))
	for i := 2; i < 16; i = i + 2 {
		result = result + ":" + fmt.Sprintf("%02x", int(ip[i]))
		result = result + fmt.Sprintf("%02x", int(ip[i+1]))
	}
	return "[" + result + "]"
}

// encode int to []byte
func encodeInt(val int) []byte {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, uint32(val))
	return buf
}

// decode []byte to int
func decodeInt(bytes []byte) int {
	return int(binary.LittleEndian.Uint32(bytes))
}
