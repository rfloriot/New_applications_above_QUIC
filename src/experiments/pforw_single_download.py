#!/usr/bin/python
from __future__ import print_function

from pyDOE import *

from helpers import *

BW_MIN = 1000
BW_MAX = 100000
FSIZES = ['8k', '64k', '256k', '1024k', '8192k']
DELAY_MIN = 10
DELAY_MAX = 100
LOSS_MIN = 0
LOSS_MAX = 2
CONCURRENCY = 1

NUM_TESTS = 200

doe_grid = lhs(4, samples=NUM_TESTS, criterion='center')


def test(net, endpoint, fsize):
    res = net.get('h1').cmd('ab -n 30 -c 1 {}/files/{} | grep Total:'.format(endpoint, fsize))
    res = res.replace('Total:', '').strip()
    return res.split()


if __name__ == "__main__":
    sys.stdout = StdoutLogger("port_forwarding/results")

    banner("system description")
    describe_system()

    banner("prerequisites")
    check("busybox", "which busybox")
    check("apache benchmark", "which ab")
    check("use cubic", "sysctl net/ipv4/tcp_congestion_control | grep cubic")
    print("[TODO: define other prerequisites]")

    banner("build network")
    net = make_network()

    banner("make ports forwarding")
    make_port_forwarding(net)

    banner("activate httpd server on h2")
    net.get('h2').cmd('busybox httpd -f -p 8080 &')
    time.sleep(1)

    banner("make experiment")

    # params = bw, fsize, loss, delay
    index = 0
    for bw_mul, fsize_mul, loss_mul, delay_mul in doe_grid:
        bw = int(BW_MIN + bw_mul * (BW_MAX - BW_MIN))
        fsize = FSIZES[int(len(FSIZES) * fsize_mul)]
        loss = int(100 * (LOSS_MIN + loss_mul * (LOSS_MAX - LOSS_MIN))) / 100.0
        delay = int(DELAY_MIN + delay_mul * (DELAY_MAX - DELAY_MIN))
        percent_completed = index/ float(20)
        banner('[{}%](bw, fsize, loss, delay) {} {} {} {}'.format(percent_completed, bw, fsize, loss, delay))
        limit_network(net, str(bw) + 'kbit', str(delay) + 'ms', 0, str(loss) + '%')

        for endpoint in ["7.7.7.7:8080", "127.0.0.1:1111", "127.0.0.1:2222", "127.0.0.1:3333"]:
            print(test(net, endpoint, fsize)[3], end=";")
        print("")
        index += 1

    banner("clean")
    net.stop()
    subprocess.call(["mn", "-c"], stdout=open(os.devnull, 'wb'), stderr=open(os.devnull, 'wb'))
