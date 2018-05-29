package main

import (
	"quic_utils"
	"crypto/rsa"
	"io/ioutil"
	"bytes"
	"strings"
	"encoding/pem"
)

// check if remote public key is in list of authorized keys
func checkClientPublicKey(s *SSHServer, receivedKey *rsa.PublicKey) bool {
	if s.conf.authorizedPublicKeysFile != "" {
		authorizedKeys := getAuthorizedKeys(s.conf.authorizedPublicKeysFile)
		i := 0
		found := false
		for i < len(authorizedKeys) && !found {
			if quic_utils.ComparePublicKeys(receivedKey, authorizedKeys[i]) {
				found = true
			}
			i = i + 1
		}
		return found
	}
	return true
}

// get list of public keys allowed, given a file listing them
func getAuthorizedKeys(file_path string) []*rsa.PublicKey {
	var result []*rsa.PublicKey = nil
	data, err := ioutil.ReadFile(file_path)
	quic_utils.Check(err)
	parts := bytes.Split(data, []byte("\n"))
	for _, line := range parts {
		if len(line) > 0 && !strings.HasPrefix(string(line), "--") {
			key := addPemMarkers(line)
			pemData, _ := pem.Decode(key)
			if pemData != nil {
				asnData := pemData.Bytes
				pk, err := quic_utils.DecodePublicKey(asnData)
				quic_utils.Check(err)
				result = append(result, pk)
			}
		}
	}
	return result
}

// add necessary fields before giving the key to Decode function.
func addPemMarkers(key []byte) []byte {
	nbr_line, i := (len(key)/64)+1, 0
	result := []byte("-----BEGIN RSA PUBLIC KEY-----\n")
	for i < nbr_line {
		if i == nbr_line-1 {
			result = append(append(result, key[i*64:]...), []byte("\n")...)
		} else {
			result = append(append(result, key[i*64:(i+1)*64]...), []byte("\n")...)
		}
		i = i + 1
	}
	result = append(result, []byte("-----END RSA PUBLIC KEY-----\n")...)
	return result
}

// undo the addPemMarkers result: get single-line string key
func removePemMarkers(key [] byte) string {
	input := string(key)
	parts := strings.Split(input, "\n")
	result := ""
	for _, part := range parts {
		if len(part) > 0 && !strings.Contains(part, "-----") {
			result = result + part
		}
	}
	return result
}
