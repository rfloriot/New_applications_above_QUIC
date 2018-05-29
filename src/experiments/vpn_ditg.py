#!/usr/bin/python
from __future__ import print_function
from helpers import *
import sys, time
from pyDOE import *
import os

BW_MIN = 1000
BW_MAX = 100000
DELAY_MIN = 10
DELAY_MAX = 100
LOSS_MIN = 0
LOSS_MAX = 2
CONCURRENCY = 4
NUM_TESTS = 100
NUM_ITERATIONS = 16 # important to avoid too much variation

BW_MIN, BW_MAX = 10000, 10000 
DELAY_MIN,DELAY_MAX = 50,50
LOSS_MIN,LOSS_MAX=0.1,0.1

doe_grid = lhs(3, samples=NUM_TESTS, criterion='center')


def test(net, endpoint):
    os.system('rm -f /tmp/itg*')

    with open('configuration/itg_script', 'r') as config:
        content = config.read()
        with open('/tmp/itg_script', 'w+') as new_config:
            new_config.write(content.replace('ENDPOINT', endpoint.replace('_ecn', '')))

    net.get('h2').cmd('iperf3 -s &')
    net.get('h1').cmd('iperf3 -c {} -t 50 > /tmp/iperf_log'.format(endpoint.replace('_ecn', '')))

    time.sleep(10)

    net.get('h2').cmd('/opt/itg/ITGRecv &')
    net.get('h1').cmd('/opt/itg/ITGSend /tmp/itg_script -l /tmp/itg_sendlog -x /tmp/itg_recvlog')

    time.sleep(15)

    log = net.get('h1').cmd('/opt/itg/ITGDec /tmp/itg_recvlog')

    net.get('h2').cmd('killall ITGRecv')
    net.get('h2').cmd('killall iperf3')
    net.get('h1').cmd('killall ITGSend')
    net.get('h1').cmd('killall ITGDec')

    res = {}
    lines = log.split('\n')
    for line in lines:
        line = line.replace(' s', '')

        if 'Flow number: ' in line:
            flow = line.strip().replace('Flow number: ', '').strip()

            if not flow in res:
                res[flow] = {}

        if '=' in line:
            parts = line.split('=')
            a, b = parts
            a = a.strip()
            b = b.strip()

            if 'delay' in a or 'jitter' in a:
                res[flow][a] = int(1000*float(b))


    for i in range(1,5):
        yield str(res[str(i)]['Average delay'])


if __name__ == "__main__":
    sys.stdout = StdoutLogger()

    banner("system description")
    describe_system()

    banner("prerequisites")
    check("openvpn", "which openvpn")
    check("iperf3", "which iperf3")
    check("apache benchmark", "which ab")
    check("use cubic", "sysctl net/ipv4/tcp_congestion_control | grep cubic")

    banner("build network")
    net = make_network()

    banner("make tunnels")
    make_tunnels(net)

    for bw_mul, loss_mul, delay_mul in doe_grid:
        bw = int(BW_MIN + bw_mul * (BW_MAX - BW_MIN))
        loss = int(100 * (LOSS_MIN + loss_mul * (LOSS_MAX - LOSS_MIN))) / 100.0
        delay = int(DELAY_MIN + delay_mul * (DELAY_MAX - DELAY_MIN))

        banner('(bw, loss, delay) {} {} {}'.format(bw, loss, delay))
        limit_network(net, str(bw) + 'kbit', str(delay) + 'ms', 0, str(loss) + '%')

        for endpoint in ["7.7.7.7", "192.168.0.2", "192.168.3.2",  "192.168.1.2_ecn", "192.168.2.2_ecn"]:
            if "_ecn" in endpoint:
                endpoint = endpoint.replace("_ecn", "")
                enableECN(net)
            else:
                disableECN(net)


            print(";".join(test(net, endpoint)), end=";;;")
        print("")

    banner("clean")
    net.stop()
