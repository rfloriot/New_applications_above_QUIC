#!/usr/bin/python
from __future__ import print_function
from helpers import *
import sys
from pyDOE import *

BW_MIN = 1000
BW_MAX = 100000
FSIZES = ['2M']
DELAY_MIN = 10
DELAY_MAX = 100
LOSS_MIN = 0
LOSS_MAX = 2
CONCURRENCY = 1 
NUM_TESTS = 100
NUM_ITERATIONS = 4 # important to avoid too much variation

doe_grid = lhs(4, samples=NUM_TESTS, criterion='center')

def test(net, endpoint, fsize):
    res = net.get('h1').cmd('ab -n {} -c {} {}/files/{} | grep Total:'.format(NUM_ITERATIONS, CONCURRENCY, endpoint, fsize))
    res = res.replace('Total:', '').strip()
    return res.split()


if __name__ == "__main__":
    sys.stdout = StdoutLogger()

    banner("system description")
    describe_system()

    banner("prerequisites")
    check("openvpn", "which openvpn")
    check("busybox", "which busybox")
    check("apache benchmark", "which ab")
    check("use cubic", "sysctl net/ipv4/tcp_congestion_control | grep cubic")

    banner("build network")
    net = make_network()
    net.get('h2').cmd('busybox httpd -f -p 80 &')

    banner("make tunnels")
    make_tunnels(net)

    for bw_mul, fsize_mul, loss_mul, delay_mul in doe_grid:
        bw = int(BW_MIN + bw_mul * (BW_MAX - BW_MIN))
        fsize = FSIZES[int(len(FSIZES) * fsize_mul)]
        loss = int(100 * (LOSS_MIN + loss_mul * (LOSS_MAX - LOSS_MIN))) / 100.0
        delay = int(DELAY_MIN + delay_mul * (DELAY_MAX - DELAY_MIN))

        banner('(bw, fsize, loss, delay) {} {} {} {}'.format(bw, fsize, loss, delay))
        limit_network(net, str(bw) + 'kbit', str(delay) + 'ms', 0, str(loss) + '%')

        for endpoint in ["7.7.7.7", "192.168.0.2", "192.168.3.2",  "192.168.2.2", "192.168.2.2_ecn"]:
            if "_ecn" in endpoint:
                endpoint = endpoint.replace("_ecn", "")
                enableECN(net)
            else:
                disableECN(net)
            print(";".join(test(net, endpoint, fsize)), end=";;;")
        print("")

    banner("clean")
    net.stop()
