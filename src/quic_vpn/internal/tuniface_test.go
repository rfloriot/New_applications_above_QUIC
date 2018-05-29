package internal

import "testing"

func TestNewTunnelIface_valid(t *testing.T){
	_, err := NewTunnelInterface(&VpnConfig{
		Iface_name: "aa",
		Iface_type: "tun",
		Ip: "192.168.0.0/24",
		Mtu:1500,
	})
	if err != nil {
		t.Errorf("Unable to create tunnel interface %s", err)
	}
}

func TestNewTunnelIface_invalid(t *testing.T){
	_, err := NewTunnelInterface(&VpnConfig{
		Iface_name: "bb",
		Iface_type: "INVALID",
		Ip: "192.168.0.0/24",
		Mtu:1500,
	})
	if err == nil {
		t.Errorf("Expected to fail creating interface")
	}

	_, err = NewTunnelInterface(&VpnConfig{
		Iface_name: "cc",
		Iface_type: "tun",
		Ip: "192.168.0.0/55",
		Mtu:1500,
	})
	if err == nil {
		t.Errorf("Expected to fail creating interface")
	}


	_, err = NewTunnelInterface(&VpnConfig{
		Iface_name: "dd",
		Iface_type: "tun",
		Ip: "192.168.0.f",
		Mtu:1500,
	})
	if err == nil {
		t.Errorf("Expected to fail creating interface")
	}


	_, err = NewTunnelInterface(&VpnConfig{
		Iface_name: "ee",
		Iface_type: "tun",
		Ip: "192.168.0.0",
		Mtu:0,
	})
	if err == nil {
		t.Errorf("Expected to fail creating interface")
	}
}