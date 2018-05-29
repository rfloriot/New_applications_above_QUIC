#!/usr/bin/python3

import os, sys, subprocess
from os.path import *


# build configuration
def generic_config(mode, multi, this_ip):
    keys = {
        'mode': mode,
        'multi_streams': multi,
        'ip': this_ip + '/24',
        'verbose': 'true',
        'iface_name': 'tun'
    }

    ret = ""
    for k, v in keys.items():
        ret += "{0}: {1}\n".format(k, v)

    return ret


def client_part(mode, keys):
    public_key, private_key = keys

    ret = "client: \n"
    keys = {'public': '"' + public_key + '"'}

    if mode == "client":
        keys['private'] = '"' + private_key + '"'
    if mode == "server":
        keys['check_key'] = 'true'

    for k, v in keys.items():
        ret += "  {0}: {1}\n".format(k, v)
    return ret


def server_part(mode, port, address, keys):
    public_key, private_key = keys

    ret = "server: \n"
    keys = {'public': '"' + public_key + '"', 'port': port}

    if mode == "server":
        keys['private'] = '"' + private_key + '"'
    if mode == "client":
        keys['check_key'] = 'true'
        keys['addr'] = address

    for k, v in keys.items():
        ret += "  {0}: {1}\n".format(k, v)
    return ret


# set up network

def main():
    if os.getuid() != 0:
        print("This script must be run under sudo user")
        sys.exit(1)

    # ----- make config -----

    # ask
    multi = input("multi-stream(true/false): ")
    mode = input("client/server: ")
    certs_dir = input('certificates directory: ')

    # generate
    server_address, server_ip = None, None

    if mode == "client":
        server_ip = input("distant ip: ")
        server_address = server_ip
    port = "4242"

    client_keys = (join(certs_dir, 'client.pub'), join(certs_dir, 'client'))
    server_keys = (join(certs_dir, 'server.pub'), join(certs_dir, 'server'))

    this_ip = "10.1.1." + ("1" if mode == "client" else "2")
    other_ip = "10.1.1." + ("2" if mode == "client" else "1")

    ret = ""
    ret += generic_config(mode, multi, this_ip)
    ret += client_part(mode, client_keys)
    ret += server_part(mode, port, server_address, server_keys)

    # write config
    with open('/tmp/vpnconfig', 'w+') as f:
        f.write(ret)

    # ----- start tunnel -----

    tunneled_ip = input("ip redirected to tunnel: ")

    print("Don't forget to run: "+"ip route add {} via {}".format(tunneled_ip, other_ip))

    os.system("./quicvpn /tmp/vpnconfig")

    popen = subprocess.Popen("sudo nohup ./quicvpn /tmp/vpnconfig 1> /tmp/vpnout 2> /tmp/vpnerr &", shell=True)
    print(popen.pid)
    print("Redirecting output to /tmp/vpnout error to /tmp/vpnerr")



if __name__ == '__main__':
    main()
