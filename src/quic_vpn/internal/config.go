// Configuration given in the command line arguments
// By CLAREMBEAU Alexis & FLORIOT Remi
// 2017-2018 Master's thesis. All rights reserved.

package internal

import (
	"fmt"
	"github.com/go-yaml/yaml"
	"io/ioutil"
	"os"
)

type VpnConfig struct {
	Mode          string
	Ip            string
	Mtu           int
	Iface_type    string
	Iface_name    string
	Multi_streams bool

	Client struct {
		Public    string
		Private   string
		Check_key bool
	}
	Server struct {
		Public    string
		Private   string
		Check_key bool
		Addr      string
		Port      int
	}
}

func (c *VpnConfig) Parse() error {
	if len(os.Args) != 2 {
		fmt.Println("usage: ./quic_vpn CONFIG")
		fmt.Println("start a VPN over QUIC with given YAML VpnConfig file")
		os.Exit(1)
	}

	filename := os.Args[1]
	configBytes, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(configBytes, c)
	return err
}
