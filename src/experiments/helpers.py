#!/usr/bin/python
from __future__ import print_function

import datetime
import subprocess
from mininet.cli import *
from mininet.net import *
from mininet.node import CPULimitedHost
from mininet.link import TCLink
from os.path import *

__dir__ = dirname(realpath(__file__))
__parent__ = dirname(__dir__)
__experiment__ = sys.argv[0].split('/')[-1].replace('.py', '').replace('experiment', '').replace('exp', '')


def make_network():
    net = Mininet(host=CPULimitedHost, link=TCLink)

    h1 = net.addHost('h1', cpu=.3)
    h2 = net.addHost('h2', cpu=.3)
    r1 = net.addHost('r1', cpu=.1)
    r2 = net.addHost('r2', cpu=.1)

    net.addLink(h1, r1)
    net.addLink(r1, r2)
    net.addLink(r2, h2)

    net.start()

    h1.setIP('6.6.6.6', 24, 'h1-eth0')
    r1.setIP('6.6.6.1', 24, 'r1-eth0')
    r1.setIP('10.0.0.2', 24, 'r1-eth1')
    r2.setIP('10.0.0.3', 24, 'r2-eth0')
    r2.setIP('7.7.7.1', 24, 'r2-eth1')
    h2.setIP('7.7.7.7', 24, 'h2-eth0')
    r1.cmd('ip route add default via 10.0.0.3')
    r2.cmd('ip route add default via 10.0.0.2')
    h1.cmd('ip route add default via 6.6.6.1')
    h2.cmd('ip route add default via 7.7.7.1')

    r1.cmd('sysctl -w net.ipv4.ip_forward=1')
    r2.cmd('sysctl -w net.ipv4.ip_forward=1')
    h1.cmd('ethtool -K h1-eth0 tx off sg off tso off')
    h2.cmd('ethtool -K h2-eth0 tx off sg off tso off')

    enableECN(net)

    net.pingAll()

    return net


def enableECN(net):
    h1, h2 = net.get('h1'), net.get('h2')
    h1.cmd('sysctl -w net.ipv4.tcp_ecn=1')
    h2.cmd('sysctl -w net.ipv4.tcp_ecn=1')


def disableECN(net):
    h1, h2 = net.get('h1'), net.get('h2')
    h1.cmd('sysctl -w net.ipv4.tcp_ecn=0')
    h2.cmd('sysctl -w net.ipv4.tcp_ecn=0')


def check(name, command):
    print(name, ":", end="")
    command = command + " 1> /dev/null 2> /dev/null"
    try:
        subprocess.check_output(command, shell=True)
        print("." * (30 - len(name)), "ok")
    except:
        print("." * (30 - len(name)), "ko!")
        sys.exit(1)


def make_tunnels(net):
    h1, h2 = net.get('h1'), net.get('h2')

    h1.cmd('killall openvpn')
    h1.cmd('killall quicvpn')
    h2.cmd('killall openvpn')
    h2.cmd('killall quicvpn')

    openvpn = "openvpn {}"
    quicvpn = join(__parent__, "quic_vpn", "quicvpn") + " {}"

    cmds = [
        (h1, openvpn.format(join(__dir__, "configuration", "openvpn_client_192.168.0.1")) + " 1> /tmp/vpn_clitcp1 2> /tmp/vpn_clitcp2 & "),
        (h2, openvpn.format(join(__dir__, "configuration", "openvpn_server_192.168.0.2")) + " 1> /tmp/vpn_servtcp1 2> /tmp/vpn_servtcp2 & "),
        (h1, quicvpn.format(join(__dir__, "configuration", "quicvpn_client_192.168.1.1")) + " 1> /tmp/vpn_cliquic1 2> /tmp/vpn_cliquic2 & "),
        (h2, quicvpn.format(join(__dir__, "configuration", "quicvpn_server_192.168.1.2")) + " 1> /tmp/vpn_servquic1 2> /tmp/vpn_servquic2 & "),
        (h1, quicvpn.format(join(__dir__, "configuration", "quicvpn_client_192.168.2.1")) + " 1> /tmp/vpn_clisquic1 2> /tmp/vpn_clisquic2 & "),
        (h2, quicvpn.format(join(__dir__, "configuration", "quicvpn_server_192.168.2.2")) + " 1> /tmp/vpn_servsquic1 2> /tmp/vpn_servsquic2 & "),

        (h1, openvpn.format(join(__dir__, "configuration",
                                 "openvpn_client_192.168.3.1")) + " 1> /tmp/vpn_cliudp1 2> /tmp/vpn_cliudp2 & "),
        (h2, openvpn.format(join(__dir__, "configuration",
                                 "openvpn_server_192.168.3.2")) + " 1> /tmp/vpn_servudp1 2> /tmp/vpn_servudp2 & "),
        (h1, 'sleep 5'),
        (h1, 'echo -n "  TCP: " ; ping 192.168.0.2 -c 1 | tail -n 1'),
        (h1, 'echo -n "  UDP: " ; ping 192.168.3.2 -c 1 | tail -n 1'),
        (h1, 'echo -n " QUIC: " ; ping 192.168.1.2 -c 1 | tail -n 1'),
        (h1, 'echo -n "sQUIC: " ; ping 192.168.2.2 -c 1 | tail -n 1'),
    ]

    for host, cmd in cmds:
        print(host.cmd(cmd).strip())


def tryForwarding(path, forwarding_type, dest, port, h1, h2, net):
    h1.cmd('curl ' + dest + ':' + port + "/test > " + path + forwarding_type)
    h1.cmd("sleep 1")
    with open(path + forwarding_type, 'r') as f:
        read_data = f.read()
    if read_data == "OK\n":
        print(forwarding_type + " : OK")
    else:
        print(forwarding_type + " : KO!")


def make_port_forwarding(net):
    # port usage summary:
    # 22 used by SSH, 12345 used by quic_ssh, 8011 used by ssf
    # server listening on 7.7.7.7:8080
    # 6.6.6.6:1111 forwarded to 7.7.7.7:8080 by quic_ssh
    # 6.6.6.6:2222 forwarded to 7.7.7.7:8080 by ssh
    # 6.6.6.6:3333 forwarded to 7.7.7.7:8080 by ssf (secured socket funneling)

    h1, h2 = net.get('h1'), net.get('h2')

    quicssh = join(__parent__, "quic_ssh", "quic_ssh_0_6")
    quicssh_client = quicssh + " --pub certificates/client.pub --priv certificates/client --req certificates/known_hosts_client 7.7.7.7 12345 -L 1111:127.0.0.1:8080 -N > /dev/null &"
    quicssh_server = quicssh + " -l --pub certificates/server.pub --priv certificates/server --req certificates/authorized_keys_server 12345 > /dev/null &"
    ssh_server = "/usr/sbin/sshd -D &"
    ssh_client = "ssh test@7.7.7.7 -o StrictHostKeyChecking=no -i certificates/test_user.private -L 2222:127.0.0.1:8080 -N &"
    ssf_server = "./ssfd &"
    ssf_client = "./ssf -L 127.0.0.1:3333:127.0.0.1:8080 7.7.7.7 &"
    h2.cmd(quicssh_server)
    h2.cmd(ssh_server)
    h2.cmd("cd ssf")
    h2.cmd(ssf_server)
    h2.cmd("cd ..")
    time.sleep(1)
    h1.cmd(quicssh_client)
    h1.cmd(ssh_client)
    h1.cmd("cd ssf")
    h1.cmd(ssf_client)
    h1.cmd("cd ..")
    time.sleep(2)
    print("done")

    banner("testing established forwarding")
    h1.cmd("rm -rf " + __dir__ + "/port_forwarding/tmp/")
    h1.cmd("mkdir " + __dir__ + "/port_forwarding/tmp")
    h2.cmd("echo 'OK' > "+ __dir__ + "/port_forwarding/tmp/test")
    h2.cmd("busybox httpd -f -p 8080 -h " + __dir__ + "/port_forwarding/tmp/ &")
    time.sleep(1)
    for item, dest, port in [("test_direct", "7.7.7.7", "8080"), ("test_quicssh", "127.0.0.1", "1111"),
                               ("test_ssh", "127.0.0.1", "2222"), ("test_ssf", "127.0.0.1", "3333")]:
        tryForwarding(__dir__ + "/port_forwarding/tmp/" + item, item, dest, port, h1, h2, net)
    h1.cmd("rm -rf " + __dir__ + "/port_forwarding/tmp/")
    h2.cmd("killall busybox")
    time.sleep(1)




def banner(name):
    n = len(name)
    stars = "*" * ((60 - n) / 2)
    print("")
    print("")
    print(stars, name, stars)


def describe_system():
    print('date: ', end='')
    print(str(datetime.datetime.now()))

    print('system information: ', end='')
    os.system('uname -sr')
    print('')

    print('sysctl TCP: ')
    os.system('sysctl -A 2> /dev/null | grep net.ipv4.tcp_ | perl -e "s/net.ipv4.tcp_//" -p')
    print('')


def limit_interface(host, iface, bw, delay, jitter, loss, queuesize=None):
    print("limit", iface, ":", bw, delay, jitter, loss, queuesize)

    cmds = [
        "tc qdisc del dev {} root".format(iface),
        "tc qdisc add dev {} handle 1: root htb default 11".format(iface),
        "tc class add dev {} parent 1: classid 1:1 htb rate 1000Mbit".format(iface, bw),
        "tc class add dev {} parent 1:1 classid 1:11 htb rate {} burst 1kb".format(iface, bw),
        "tc qdisc add dev {} parent 1:11 handle 10: netem delay {} {} loss {} limit {}".format(iface, delay, jitter,
                                                                                               loss, queuesize),
    ]

    for x in cmds:
        x, host.cmd(x)

def convert_kbps(string):
    string = string.lower()
    return int(string.replace('kbps', '000').replace('kbit','000').replace('mbps', '000000').replace('mbit', '000000'))/1000

def limit_network(net, bw, delay, jitter, loss, queuesize=None):
    if queuesize is None:
        queing_delay_max = 2*int(delay.replace('ms', ''))
        best_queue = int((int(convert_kbps(bw)) / (8*1.5)) * (queing_delay_max/1000.0) * 1.5)
    else:
        print("/ ! \\")
        print("/ ! \\ Warning: custom queue size can have undesired effect")
        print("/ ! \\")
        best_queue= queuesize

    limit_interface(net.get('r1'), 'r1-eth1', bw, delay, jitter, loss, best_queue)
    limit_interface(net.get('r2'), 'r2-eth0', bw, delay, jitter, loss, best_queue)

def get_results_prefix(other_path=""):
    datestr = str(datetime.datetime.now()).split('.')[0]
    datestr = datestr.replace(' ', '_')
    fname = "{}_{}".format(__experiment__, datestr)
    if len(other_path) > 0:
        return join(__dir__, other_path, fname)
    else:
        return join(__dir__, "results", fname)


class StdoutLogger(object):
    def __init__(self, other_path=""):
        self.terminal = sys.stdout
        path = join(get_results_prefix(other_path) + "_log.csv")
        self.log = open(path, "a")

    def write(self, message):
        self.terminal.write(message)
        self.log.write(message)
        self.terminal.flush()
        self.log.flush()
