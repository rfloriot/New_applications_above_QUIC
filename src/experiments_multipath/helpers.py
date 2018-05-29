#!/usr/bin/python
from __future__ import print_function

import datetime
import subprocess
from mininet.cli import *
from mininet.net import *
from mininet.node import CPULimitedHost
from os.path import *
from mininet.link import TCLink

__dir__ = dirname(realpath(__file__))
__parent__ = dirname(__dir__)
__experiment__ = sys.argv[0].split('/')[-1].replace('.py', '').replace('experiment', '').replace('exp', '')


def make_network():
    net = Mininet(host=CPULimitedHost, link=TCLink)

    h1 = net.addHost('h1', cpu=.2)
    h2 = net.addHost('h2', cpu=.2)
    r1 = net.addHost('r1', cpu=.2)
    r2 = net.addHost('r2', cpu=.2)
    r3 = net.addHost('r3', cpu=.2)

    net.addLink(h1, r1)
    net.addLink(h1, r2)
    net.addLink(r1, r3)
    net.addLink(r2, r3)
    net.addLink(r3, h2)

    net.start()

    h1.setIP('3.3.3.3', 24, 'h1-eth0')
    h1.setIP('4.4.4.4', 24, 'h1-eth1')
    r1.setIP('3.3.3.1', 24, 'r1-eth0')
    r1.setIP('5.5.5.1', 24, 'r1-eth1')
    r2.setIP('4.4.4.1', 24, 'r2-eth0')
    r2.setIP('6.6.6.1', 24, 'r2-eth1')
    r3.setIP('5.5.5.2', 24, 'r3-eth0')
    r3.setIP('6.6.6.2', 24, 'r3-eth1')
    r3.setIP('7.7.7.1', 24, 'r3-eth2')
    h2.setIP('7.7.7.7', 24, 'h2-eth0')

    h1.cmd('ip rule add from 3.3.3.3 table 1')
    h1.cmd('ip rule add from 4.4.4.4 table 2')
    h1.cmd('ip route add 3.3.3.0/24 dev h1-eth0 scope link table 1')
    h1.cmd('ip route add default via 3.3.3.1 dev h1-eth0 table 1')
    h1.cmd('ip route add 4.4.4.0/24 dev h1-eth1 scope link table 2')
    h1.cmd('ip route add default via 4.4.4.1 dev h1-eth1 table 2')
    h1.cmd('ip route add default scope global nexthop via 3.3.3.1 dev h1-eth0')

    h2.cmd('ip route add default via 7.7.7.1')

    r1.cmd('sysctl -w net.ipv4.ip_forward=1')
    r2.cmd('sysctl -w net.ipv4.ip_forward=1')
    r3.cmd('sysctl -w net.ipv4.ip_forward=1')

    r3.cmd('ip route add 3.3.3.0/24 via 5.5.5.1 dev r3-eth0')
    r3.cmd('ip route add 4.4.4.0/24 via 6.6.6.1 dev r3-eth1')
    r1.cmd('ip route add 7.7.7.0/24 via 5.5.5.2 dev r1-eth1')
    r1.cmd('ip route add 4.4.4.0/24 via 5.5.5.2 dev r1-eth1')
    r1.cmd('ip route add 6.6.6.0/24 via 5.5.5.2 dev r1-eth1')
    r2.cmd('ip route add 7.7.7.0/24 via 6.6.6.2 dev r2-eth1')
    r2.cmd('ip route add 3.3.3.0/24 via 6.6.6.2 dev r2-eth1')
    r2.cmd('ip route add 5.5.5.0/24 via 6.6.6.2 dev r2-eth1')


    enable_ecn(net)

    net.pingAll()

    return net


def enable_ecn(net):
    h1, h2 = net.get('h1'), net.get('h2')
    h1.cmd('sysctl -w net.ipv4.tcp_ecn=1')
    h2.cmd('sysctl -w net.ipv4.tcp_ecn=1')


def disable_ecn(net):
    h1, h2 = net.get('h1'), net.get('h2')
    h1.cmd('sysctl -w net.ipv4.tcp_ecn=0')
    h2.cmd('sysctl -w net.ipv4.tcp_ecn=0')


def check(name, command):
    print(name, ":", end="")
    command = command + " 1> /dev/null 2> /dev/null"
    try:
        subprocess.check_output(command, shell=True)
        print("." * (30 - len(name)), "ok")
    except error:
        print("." * (30 - len(name)), "ko!")
        sys.exit(1)


def try_forwarding(path, forwarding_type, dest, port, h1):
    h1.cmd('curl ' + dest + ':' + port + "/test > " + path + forwarding_type)
    h1.cmd("sleep 1")
    with open(path + forwarding_type, 'r') as f:
        read_data = f.read()
    if read_data == "OK\n":
        print(forwarding_type + " : OK")
    else:
        print(forwarding_type + " : KO!")


def make_port_forwardingSingle(net):
    # port usage summary:
    # 22 used by SSH, 12345 used by quic_ssh, 8011 used by ssf
    # server listening on 7.7.7.7:8080
    # 127.0.0.1:1111 on client forwarded to 7.7.7.7:8080 by quic_ssh via path north (via r1)
    # 127.0.0.1:9999 on client forwarded to 7.7.7.7:8080 by quic_ssh over quic-multipath

    h1, h2 = net.get('h1'), net.get('h2')

    quicssh = join(__parent__, "quic_ssh", "quic_ssh_0_6")
    quicssh_multi = join(__parent__, "quic_ssh", "quic_ssh_multi")
    quicssh_client = quicssh + " --pub certificates/client.pub --priv certificates/client --req certificates/known_hosts_client 7.7.7.7 12345 -L 1111:127.0.0.1:8080 -N > /dev/null & "
    quicssh_server = quicssh + " -l --pub certificates/server.pub --priv certificates/server --req certificates/authorized_keys_server 12345 > /dev/null & "
    quicssh_multi_client = quicssh_multi + " --pub certificates/client.pub --priv certificates/client --req certificates/known_hosts_client 7.7.7.7 54321 -L 9999:127.0.0.1:8080 -N > /dev/null & "
    quicssh_multi_server = quicssh_multi + " -l --pub certificates/server.pub --priv certificates/server --req certificates/authorized_keys_server 54321 > /dev/null & "

    h2.cmd(quicssh_server)

    time.sleep(1)
    h1.cmd(quicssh_client)
    time.sleep(1)

def make_port_forwardingMulti(net):
    # port usage summary:
    # 22 used by SSH, 12345 used by quic_ssh, 8011 used by ssf
    # server listening on 7.7.7.7:8080
    # 127.0.0.1:1111 on client forwarded to 7.7.7.7:8080 by quic_ssh via path north (via r1)
    # 127.0.0.1:9999 on client forwarded to 7.7.7.7:8080 by quic_ssh over quic-multipath

    h1, h2 = net.get('h1'), net.get('h2')

    quicssh = join(__parent__, "quic_ssh", "quic_ssh_0_6")
    quicssh_multi = join(__parent__, "quic_ssh", "quic_ssh_multi")
    quicssh_client = quicssh + " --pub certificates/client.pub --priv certificates/client --req certificates/known_hosts_client 7.7.7.7 12345 -L 1111:127.0.0.1:8080 -N > /dev/null & "
    quicssh_server = quicssh + " -l --pub certificates/server.pub --priv certificates/server --req certificates/authorized_keys_server 12345 > /dev/null & "
    quicssh_multi_client = quicssh_multi + " --pub certificates/client.pub --priv certificates/client --req certificates/known_hosts_client 7.7.7.7 54321 -L 9999:127.0.0.1:8080 -N > /dev/null & "
    quicssh_multi_server = quicssh_multi + " -l --pub certificates/server.pub --priv certificates/server --req certificates/authorized_keys_server 54321 > /dev/null & "

    h2.cmd(quicssh_multi_server)

    time.sleep(1)
    h1.cmd(quicssh_multi_client)
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
    return int(
        string.replace('kbps', '000').replace('kbit', '000').replace('mbps', '000000').replace('mbit', '000000')) / 1000


def limit_network(net, bw, delay, jitter, loss, queuesize=None, north=True):
    if queuesize is None:
        queing_delay_max = 200
        best_queue = int((int(convert_kbps(bw)) / (8 * 1.5)) * (queing_delay_max / 1000.0) * 1.5)
    else:
        print("/ ! \\")
        print("/ ! \\ Warning: custom queue size can have undesired effect")
        print("/ ! \\")
        best_queue = queuesize

    if north:
        limit_interface(net.get('r1'), 'r1-eth1', bw, delay, jitter, loss, best_queue)
        limit_interface(net.get('r3'), 'r3-eth0', bw, delay, jitter, loss, best_queue)
    else:
        limit_interface(net.get('r2'), 'r2-eth1', bw, delay, jitter, loss, best_queue)
        limit_interface(net.get('r3'), 'r3-eth1', bw, delay, jitter, loss, best_queue)


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
