#!/usr/bin/python
from __future__ import print_function

from pyDOE import *

from helpers import *

BW_MIN = 1000
BW_MAX = 100000
FSIZES = ['8k', '64k', '256k', '1024k', '8192k']
DELAY_MIN = 1  # 10
DELAY_MAX = 25  # 100
LOSS_MIN = 0
LOSS_MAX = 0  # 2
CONCURRENCY = 1

NUM_TESTS = 200

doe_grid = lhs(4, samples=NUM_TESTS, criterion='center')
grid_2nd_path = lhs(3, samples=NUM_TESTS, criterion='center')


def test(net, endpoint, fsize):
    res = net.get('h1').cmd('ab -n 30 -c 1 {}/files/{} | grep Total:'.format(endpoint, fsize))
    res = res.replace('Total:', '').strip()
    return res.split()


def setupSingle(net):
    net.get('h1').cmd("pkill quic_ssh_0.6")
    net.get('h2').cmd("pkill quic_ssh_0.6")
    net.get('h1').cmd("pkill quic_ssh_multi")
    net.get('h2').cmd("pkill quic_ssh_multi")
    time.sleep(1)
    make_port_forwardingSingle(net)
    time.sleep(1)


def setupMulti(net):
    net.get('h1').cmd("pkill quic_ssh_0.6")
    net.get('h2').cmd("pkill quic_ssh_0.6")
    net.get('h1').cmd("pkill quic_ssh_multi")
    net.get('h2').cmd("pkill quic_ssh_multi")
    time.sleep(1)
    make_port_forwardingMulti(net)
    time.sleep(1)

if __name__ == "__main__":
    sys.stdout = StdoutLogger("port_forwarding/results")

    banner("system description")
    describe_system()

    banner("prerequisites")
    check("busybox", "which busybox")
    check("apache benchmark", "which ab")
    check("use cubic", "sysctl net/ipv4/tcp_congestion_control | grep cubic")

    banner("build network")
    net = make_network()

    banner("make experiment")
    net.get('h2').cmd('busybox httpd -f -p 8080 &')

    # params = bw, fsize, loss, delay
    index = 0
    for bw_mul, fsize_mul, loss_mul, delay_mul in doe_grid:
        bw = int(BW_MIN + bw_mul * (BW_MAX - BW_MIN))
        fsize = FSIZES[int(len(FSIZES) * fsize_mul)]
        loss = int(100 * (LOSS_MIN + loss_mul * (LOSS_MAX - LOSS_MIN))) / 100.0
        delay = int(DELAY_MIN + delay_mul * (DELAY_MAX - DELAY_MIN))

        bw_mul2, loss_mul2, delay_mul2 = grid_2nd_path[index]
        bw2 = int(BW_MIN + bw_mul2 * (BW_MAX - BW_MIN))
        loss2 = int(100 * (LOSS_MIN + loss_mul2 * (LOSS_MAX - LOSS_MIN))) / 100.0
        delay2 = int(DELAY_MIN + delay_mul2 * (DELAY_MAX - DELAY_MIN))

        percent_completed = index / float(2)

        banner(
            '[{}%] (fsize, bw, loss, delay, bw2, loss2, delay2) {} {} {} {} {} {} {}'.format(percent_completed, fsize,
                                                                                             bw, loss, delay, bw2,
                                                                                             loss2, delay2))
        limit_network(net, str(bw) + 'kbit', str(delay) + 'ms', 0, str(loss) + '%', north=True)
        limit_network(net, str(bw) + 'kbit', str(delay) + 'ms', 0, str(loss) + '%', north=False)

        setupSingle(net)

        print(test(net, "127.0.0.1:1111", fsize)[3], end=";")

        limit_network(net, str(bw2) + 'kbit', str(delay2) + 'ms', 0, str(loss2) + '%', north=True)
        limit_network(net, str(bw2) + 'kbit', str(delay2) + 'ms', 0, str(loss2) + '%', north=False)

        print(test(net, "127.0.0.1:1111", fsize)[3], end=";")

        limit_network(net, str(bw) + 'kbit', str(delay) + 'ms', 0, str(loss) + '%', north=True)

        setupMulti(net)

        print(test(net, "127.0.0.1:9999", fsize)[3], end=";")
        print("")
        index += 1

    banner("clean")
    net.stop()
    subprocess.call(["mn", "-c"], stdout=open(os.devnull, 'wb'), stderr=open(os.devnull, 'wb'))
