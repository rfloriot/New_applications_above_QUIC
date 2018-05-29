package main

import (
	"os/exec"
	"os"
	"fmt"
	"testing"
	"strings"
	"time"
)

const dummyServerPublicKeyInline = "MIGJAoGBAKjbx1uVtvN+i/W+HivtuHPDlsvlY4GUO3IUCqVvmGaunlXeYNJSdki4+BSbmX7oiwRkIIhBYSHTWfeiSzbBKZGarHy4lXXQSJdzVSD5qS3AfZvPtWHZhQRGW8LtEtmohk90qGkQWkfgHmD3Zrz9I+JqECq53g9DBmHRcvbzeDuVAgMBAAE="
const dummyClientPublicKeyInline = "MIGJAoGBAMlBdZvARrLyVK5B8yyojAKB0f70RSauEqxVvZ9mGbI+J/dWFQZmjILrWtvw8mcfsLYLIq6XD1WUjJP+CfulY/C2WOZxCUeL0rTophtcNx3lgPX4G4rRza8zhMKPjDBCjoWbxCEfoPwQG4eeJh2w18cSspx1NmSIpv/dsSo5ViVhAgMBAAE="
const dummyServerPublicKey = "-----BEGIN RSA PUBLIC KEY-----\nMIGJAoGBAKjbx1uVtvN+i/W+HivtuHPDlsvlY4GUO3IUCqVvmGaunlXeYNJSdki4\n+BSbmX7oiwRkIIhBYSHTWfeiSzbBKZGarHy4lXXQSJdzVSD5qS3AfZvPtWHZhQRG\nW8LtEtmohk90qGkQWkfgHmD3Zrz9I+JqECq53g9DBmHRcvbzeDuVAgMBAAE=\n-----END RSA PUBLIC KEY-----\n"
const dummyClientPublicKey = "-----BEGIN RSA PUBLIC KEY-----\nMIGJAoGBAMlBdZvARrLyVK5B8yyojAKB0f70RSauEqxVvZ9mGbI+J/dWFQZmjILr\nWtvw8mcfsLYLIq6XD1WUjJP+CfulY/C2WOZxCUeL0rTophtcNx3lgPX4G4rRza8z\nhMKPjDBCjoWbxCEfoPwQG4eeJh2w18cSspx1NmSIpv/dsSo5ViVhAgMBAAE=\n-----END RSA PUBLIC KEY-----\n"
const dummyServerPrivateKey = "-----BEGIN RSA PRIVATE KEY-----\nMIICXAIBAAKBgQCo28dblbbzfov1vh4r7bhzw5bL5WOBlDtyFAqlb5hmrp5V3mDS\nUnZIuPgUm5l+6IsEZCCIQWEh01n3oks2wSmRmqx8uJV10EiXc1Ug+aktwH2bz7Vh\n2YUERlvC7RLZqIZPdKhpEFpH4B5g92a8/SPiahAqud4PQwZh0XL283g7lQIDAQAB\nAoGARP1Gfky06tcRJ939RcViTyniOnwGI7MEdp9pmh32Dj3ZwwuQU14Npbis4v6P\nwCISakDeac0Mel13rI1KXZyd9o0jIUexJ7R+TJn3bxnE1r2XzQYv19z4pRBV8TRi\nieMLGAGYoztVvOtFp/hLmnOHhUVkQLREyeya8HsDRS2xUCECQQDchnoI9NNn+RZg\nbnHlVki0VH/iw2jKuBvPdftszjMjNtEJBZCad39xhKvpswkyel2e9Fht1WRl0sYW\n3zjqs1zbAkEAxAWYa2DZusgwdGH4Q+/qm5YnV14kr35WSh/RWHWeSIhe3PuZoI5W\n6qMB8SNsNWsZIUtWdeVZFjbFqbbZBlX8TwJBAK5mN1qf7BS8+8JVdgON4j+i1+SI\n73XqdivyvV0GEZEWx+ffm8VdHc+zwZU3ft2JwkJ0MP7jlNul/fyWmleac6MCQBnq\nx3lDB+abO1TX8zRAT1uc4by6dM1DPfN0+3/fpTrf1PMQzQIeb718KfCRB2iUrXDq\nfhb+aOX3/fBvfYhJ7B8CQE5PvjXPSH+79pC+1x25lExILg9IkIoc1fibhCX9ltIb\nkYpt8KMYkAzwS7zFlGCxOwoV75ccHpKojH7wKpVGwIY=\n-----END RSA PRIVATE KEY-----"
const dummyClientPrivateKey = "-----BEGIN RSA PRIVATE KEY-----\nMIICXQIBAAKBgQDJQXWbwEay8lSuQfMsqIwCgdH+9EUmrhKsVb2fZhmyPif3VhUG\nZoyC61rb8PJnH7C2CyKulw9VlIyT/gn7pWPwtljmcQlHi9K06KYbXDcd5YD1+BuK\n0c2vM4TCj4wwQo6Fm8QhH6D8EBuHniYdsNfHErKcdTZkiKb/3bEqOVYlYQIDAQAB\nAoGBAKG6OcF8xROeS1BxbPIRS9nj6xX/w+YucpEMocILMVEcQ8+t3F11YSr/6Nbg\nDFu0irPvxOIaQFdcdY+j0O/pW6IyCs7gnG69fztAf7StRUx2NVTQsTetmoPBl1H1\nVvZ0l42UXvYniE9Bu55MN0CWAcCyCgLjMeSJ43IB+YvvKxP1AkEA141q40UecQQ4\na+wBDv9EiGd8HeXOf/sfcM14GwAmaZC9OXNYBquG5l94NCyaFH5mMuAa0lsYBWeC\n7q/wPEDjwwJBAO8FQ8NWPTC6vNPb6hYa8Ulf+8FcWpFJIJETcjvYSGth76kctUC6\ngqldBPz0z0i/30nJaXd5W42su5G9mNnIdAsCQF5a+T8jMoAmaMxVMuFtvII5SouL\n3SkItGqchsbK+gWb5jkP1KiWzSZrBCNSot/1tKbwks0iMxGqjhYNzguSHCECQGbZ\nNRdQfHQDZk0jS87HORwBmSrSuoXZmZHTdEwb/M14DtAN8lAv8Rk/VW4jSS5coY/2\ngtNN/P8xXGSR2LudbZECQQCS+TtTfYM29vY0HpnWM8jkolzns7e9Qj/BlMakg8nB\nB3LAx0A+v+5klf0dqzBbOthZJS0q2QJK89ghUi8fkZAu\n-----END RSA PRIVATE KEY-----"
const directory = "/tmp/test_quicssh_go/"

func initTests() {
	cmd0 := exec.Command("rm", "-rf", "test_quicssh_go")
	cmd0.Dir = "/tmp"
	cmd0.Run()

	cmd := exec.Command("mkdir", "test_quicssh_go")
	cmd.Dir = "/tmp"
	cmd.Run()

	writeFile(directory+"known_hosts_client_void", "")
	writeFile(directory+"known_hosts_client_with_invalid_key",
		"127.0.0.1:5050 " + dummyServerPublicKeyInline[:len(dummyClientPublicKeyInline)-1] + "@\n"+ // add invalid key
			"127.0.0.1:5050 "+ dummyServerPublicKeyInline+ "\n")
	writeFile(directory+"authorized_hosts_server", dummyClientPublicKeyInline+"\n")
	writeFile(directory+"pk_client", dummyClientPublicKey)
	writeFile(directory+"pk_server", dummyServerPublicKey)
	writeFile(directory+"pr_client", dummyClientPrivateKey)
	writeFile(directory+"pr_server", dummyServerPrivateKey)
	writeFile(directory+"authorized_hosts_server", dummyClientPublicKeyInline)
	writeFile(directory+"log", "")
}

func init() {
	initTests()
	logTmp("1")
}

func logTmp(str string) {
	now := time.Now()
	ye, mo, da := now.Date()
	h, m, s := now.Hour(), now.Minute(), now.Second()
	nowStr := fmt.Sprintf("[%04d/%02d/%02d - %02d:%02d:%02d]", ye, mo, da, h, m, s)
	appendToFile(directory+"log", nowStr+"\n"+str+"\n")
}

func appendToFile(file string, content string) {
	f, err1 := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0600)
	line := content
	n, err2 := f.Write([]byte(line))
	err3 := f.Sync()
	err4 := f.Close()
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || n != len(line) {
		fmt.Printf("Cannot launch tests: unable to write dummy data inside /tmp/test_quicssh_go/")
		os.Exit(-1)
	}
}

func writeFile(file string, content string) {
	f, err1 := os.OpenFile(file, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
	line := content
	n, err2 := f.Write([]byte(line))
	err3 := f.Sync()
	err4 := f.Close()
	if err1 != nil || err2 != nil || err3 != nil || err4 != nil || n != len(line) {
		fmt.Printf("Cannot launch tests: unable to write dummy data inside /tmp/test_quicssh_go/")
		os.Exit(-1)
	}
}
func boolToString(b bool) (string) {
	if b {
		return "true"
	} else {
		return "false"
	}
}

func checkValueInt(what string, shouldBe int, is int, t *testing.T) {
	if shouldBe != is {
		t.Errorf("Error: %s should be equal to %d but is %d\n", what, shouldBe, is)
	}
}
func checkValueString(what string, shouldBe string, is string, t *testing.T) {
	if shouldBe != is {
		t.Errorf("Error: %s should be equal to '%s' but is '%s'\n", what, shouldBe, is)
	}
}
func checkValueBoolean(what string, shouldBe bool, is bool, t *testing.T) {
	if shouldBe != is {
		t.Errorf("Error: %s should be equal to '%s' but is '%s'\n", what, boolToString(shouldBe), boolToString(is))
	}
}

func checkPresenceOfUsage(command string, t *testing.T) {
	os.Args = strings.Split(command, " ")
	conf := SSHConfig{}
	conf.testMode = true
	conf.parseArguments()
	checkValueBoolean("'presence of usage printed'", true, strings.Contains(conf.testOutput, "Usage:"), t)
}

func TestArgumentParsing(t *testing.T) {
	//successful commands:
	command := "quic_ssh --pub ../quic_utils/certs/client.pub --priv ../quic_utils/certs/client --req known_hosts_client 127.0.0.1 5050 -N -L 1234:127.0.0.1:5678"
	os.Args = strings.Split(command, " ")
	conf := SSHConfig{}
	conf.testMode = true
	conf.parseArguments()
	checkValueString("public key file", "../quic_utils/certs/client.pub", conf.pubKeyFile, t)
	checkValueString("private key file", "../quic_utils/certs/client", conf.privKeyFile, t)
	checkValueString("known hosts file", "known_hosts_client", conf.authorizedPublicKeysFile, t)
	checkValueString("server address", "127.0.0.1", conf.hostname, t)
	checkValueString("hostname forwarding", "127.0.0.1", ipToString(conf.remoteIP), t)
	checkValueInt("server port", 5050, conf.port, t)
	checkValueInt("local port", 1234, int(conf.localPort), t)
	checkValueInt("remote port", 5678, int(conf.remotePort), t)
	checkValueBoolean("conf.localPortForwarding", true, conf.localPortForwarding, t)
	checkValueBoolean("conf.onlyForwardPort", true, conf.onlyForwardPort, t)
	checkValueBoolean("'absence of usage printed'", true, !strings.Contains(conf.testOutput, "usage"), t)

	command = "quic_ssh --pub ../quic_utils/certs/client.pub --priv ../quic_utils/certs/client --req known_hosts_client 127.0.0.1 5050 -R 1234:127.0.0.1:5678 --user username --pass password"
	os.Args = strings.Split(command, " ")
	conf = SSHConfig{}
	conf.testMode = true
	conf.parseArguments()
	checkValueString("public key file", "../quic_utils/certs/client.pub", conf.pubKeyFile, t)
	checkValueString("private key file", "../quic_utils/certs/client", conf.privKeyFile, t)
	checkValueString("known hosts file", "known_hosts_client", conf.authorizedPublicKeysFile, t)
	checkValueString("server address", "127.0.0.1", conf.hostname, t)
	checkValueString("hostname forwarding", "127.0.0.1", ipToString(conf.remoteIP), t)
	checkValueString("username", "username", conf.username, t)
	checkValueString("password", "password", conf.password, t)
	checkValueInt("server port", 5050, conf.port, t)
	checkValueInt("local port", 1234, int(conf.localPort), t)
	checkValueInt("remote port", 5678, int(conf.remotePort), t)
	checkValueBoolean("conf.localPortForwarding", false, conf.localPortForwarding, t)
	checkValueBoolean("conf.remotePortForwarding", true, conf.remotePortForwarding, t)
	checkValueBoolean("conf.onlyForwardPort", false, conf.onlyForwardPort, t)
	checkValueBoolean("'absence of usage printed'", true, !strings.Contains(conf.testOutput, "usage"), t)

	command = "quic_ssh -l --pub ../quic_utils/certs/server.pub --priv ../quic_utils/certs/server --req authorized_keys_server 5050"
	os.Args = strings.Split(command, " ")
	conf = SSHConfig{}
	conf.testMode = true
	conf.parseArguments()
	checkValueString("public key file", "../quic_utils/certs/server.pub", conf.pubKeyFile, t)
	checkValueString("private key file", "../quic_utils/certs/server", conf.privKeyFile, t)
	checkValueString("known hosts file", "authorized_keys_server", conf.authorizedPublicKeysFile, t)
	checkValueInt("port to listen", 5050, conf.port, t)
	checkValueBoolean("conf.listen", true, conf.listen, t)
	checkValueBoolean("'absence of usage printed'", true, !strings.Contains(conf.testOutput, "usage"), t)

	// unsuccessful commands that results in printing usage:
	checkPresenceOfUsage("quic_ssh -l --pub ../quic_utils/certs/server.pub --priv ../quic_utils/certs/server --req authorized_keys_server 5050 -L 1234:localhost:5678 -R 2345:localhost:3456", t)
	checkPresenceOfUsage("quic_ssh -l --pub ../quic_utils/certs/server.pub --priv ../quic_utils/certs/server --req authorized_keys_server 5050 too_much", t)
	checkPresenceOfUsage("quic_ssh -l --pub ../quic_utils/certs/server.pub --priv ../quic_utils/certs/server --req authorized_keys_server 5050 really too_much", t)
	checkPresenceOfUsage("quic_ssh -l --pub ../quic_utils/certs/server.pub --priv ../quic_utils/certs/server --req authorized_keys_server -N 5050", t)
	checkPresenceOfUsage("quic_ssh -l", t)
	checkPresenceOfUsage("quic_ssh", t)
	checkPresenceOfUsage("quic_ssh -h", t)
	checkPresenceOfUsage("quic_ssh -b bad_integer", t)
	checkPresenceOfUsage("quic_ssh -L A:[2001::1]:B", t)
	checkPresenceOfUsage("quic_ssh -R A:[2001::2]:B", t)
	checkPresenceOfUsage("quic_ssh -L 1234:localhost:5678 -R 2345:localhost:6789", t)
	checkPresenceOfUsage("quic_ssh -R 2345:localhost:6789 -L 1234:localhost:5678", t)
	checkPresenceOfUsage("quic_ssh -R 12:34:56:78", t)
	checkPresenceOfUsage("quic_ssh -L 12:34:56:78", t)

}
