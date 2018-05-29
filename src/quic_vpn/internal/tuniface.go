// Tun/Tap interface creation
// By CLAREMBEAU Alexis & FLORIOT Remi
// 2017-2018 Master's thesis. All rights reserved.

package internal

import (
	"errors"
	"fmt"
	"github.com/songgao/water"
	"os/exec"
)

const (
	cmdAddAddr = "ip addr add dev %v %v"
	cmdSetUp   = "ip link set dev %v up"
	cmdSetMtu  = "ip link set dev %v mtu %v qlen 100"

	debugIfaceType    = "detected interface type: %v\n"
	debugIfaceCreated = "interface created: %v\n"
)

var (
	errUnknownIface = errors.New("unknown interface type")
)

// Create a new tunnel interface from a configuration
func NewTunnelInterface(cliConfig *VpnConfig) (*water.Interface, error) {
	waterConfig, err := newWaterConfig(cliConfig)
	if err != nil {
		return nil, err
	}

	waterInterface, err := water.New(*waterConfig)
	if err != nil {
		return nil, err
	}

	if err := configureWaterInterface(waterInterface, cliConfig); err != nil {
		return nil, err
	}

	fmt.Printf(debugIfaceCreated, waterInterface.Name())
	return waterInterface, err
}

// create the new water config from VpnConfig
func newWaterConfig(cliConfig *VpnConfig) (*water.Config, error) {
	waterConfig := water.Config{}
	if err := fillInterfaceType(&waterConfig, cliConfig); err != nil {
		return nil, err
	}
	waterConfig.Name = cliConfig.Iface_name

	return &waterConfig, nil
}

// fill water type from vpnConfig string type
func fillInterfaceType(waterConf *water.Config, cliConf *VpnConfig) error {
	fmt.Printf(debugIfaceType, cliConf.Iface_type)
	if cliConf.Iface_type == "tun" {
		waterConf.DeviceType = water.TUN
	} else {
		return errUnknownIface
	}
	return nil
}

// configure interface: set ip, mtu, set up, ...
func configureWaterInterface(waterInterface *water.Interface, cliConfig *VpnConfig) error {
	commandList := []string{
		fmt.Sprintf(cmdAddAddr, waterInterface.Name(), cliConfig.Ip),
		fmt.Sprintf(cmdSetUp, waterInterface.Name()),
		fmt.Sprintf(cmdSetMtu, waterInterface.Name(), cliConfig.Mtu),
	}
	for _, cmd := range commandList {
		if stdout, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
			return errors.New(fmt.Sprintf("configureInterface: '%v' failed: '%v'", cmd, stdout))
		}
	}
	return nil
}
