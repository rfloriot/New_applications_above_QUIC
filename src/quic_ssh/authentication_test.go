package main

import (
	"quic_utils"
	"testing"
	"strings"
)

func init() {
	logTmp("2")
}

func TestCheckRemotePublicKey(t *testing.T) {
	// construct config and other args
	conf := SSHConfig{}
	conf.testMode = true
	conf.authorizedPublicKeysFile = directory + "known_hosts_client_with_invalid_key"
	remoteAddr := "127.0.0.1:5050"
	serverPk, err := quic_utils.ExtractPublicKey(directory + "pk_server")
	if err == nil {
		result, _ := checkRemotePublicKey(&conf, remoteAddr, serverPk)
		if ! result {
			t.Errorf("check return false when it should return true")
		}
		remoteAddr2 := "123.456.789.0:5050"
		result, _ = checkRemotePublicKey(&conf, remoteAddr2, serverPk)
		if result {
			t.Errorf("check return true when it should return false")
		}
		serverPk2, _ := quic_utils.ExtractPublicKey(directory + "pk_client")
		result, _ = checkRemotePublicKey(&conf, remoteAddr, serverPk2)
		if result {
			t.Errorf("check return true when it should return false")
		}
	} else {
		t.Errorf("cannot extract properly server public key from file")
	}
}

func TestAddPemMarkers(t *testing.T) {
	// write dummy key inline and check if it is correctly splitted and surrounded with pem markers
	key := dummyClientPublicKeyInline
	result := string(addPemMarkers([]byte(key)))
	parts := strings.Split(result, "\n")
	if len(parts) < 5 {
		t.Errorf("%d line expected (got %d)", 5, len(parts))
	} else if parts[0] != "-----BEGIN RSA PUBLIC KEY-----" {
		t.Errorf("result should contain start line")
	} else if parts[1] != "MIGJAoGBAMlBdZvARrLyVK5B8yyojAKB0f70RSauEqxVvZ9mGbI+J/dWFQZmjILr" ||
		parts[2] != "Wtvw8mcfsLYLIq6XD1WUjJP+CfulY/C2WOZxCUeL0rTophtcNx3lgPX4G4rRza8z" ||
		parts[3] != "hMKPjDBCjoWbxCEfoPwQG4eeJh2w18cSspx1NmSIpv/dsSo5ViVhAgMBAAE=" {
		t.Errorf("bad content")
	} else if parts[4] != "-----END RSA PUBLIC KEY-----" {
		t.Errorf("result should contain end line")
	} else if len(parts) == 6 && parts[5] != "" {
		t.Errorf("result should not contain data after end line")
	}
}

func TestRemovePemMarkers(t *testing.T) {
	// use dummy key with pem markers and extract inline key
	key1, key2 := dummyServerPublicKey, dummyClientPublicKey
	result1, result2 := string(removePemMarkers([]byte(key1))), string(removePemMarkers([]byte(key2)))
	parts1, parts2 := strings.Split(result1, "\n"), strings.Split(result2, "\n")
	if len(parts1) != 1 || len(parts2) != 1 {
		t.Errorf("result should contain only one line")
	}

	if result1 != dummyServerPublicKeyInline || result2 != dummyClientPublicKeyInline {
		t.Errorf("result not correctly constructed")
	}

}
