#!/usr/bin/python
from __future__ import print_function
from helpers import *
import sys
from pyDOE import *

TIMEOUT = 20 # in sec

if len(sys.argv) == 1:
    print('Running a single experiment')
    bw = int(input('BW: '))
    fsize = int(input('File size: ') )
    delay = int(input('Delay: ') )
    loss = int(input('Loss: ') )
    jitter = int(input('Jitter: ') )

    BW_MIN, BW_MAX = bw, bw
    FSIZES = [fsize]
    DELAY_MIN, DELAY_MAX = delay
    LOSS_MIN, LOSS_MAX = loss
    JITTER_MIN, JITTER_MAX = jitter
    NUM_TESTS = 1
    NUM_ITERATIONS = 5
else:
    mode = int(sys.argv[1])

    BW_MIN = 1000
    BW_MAX = 100000
    DELAY_MIN = 10
    DELAY_MAX = 100
    LOSS_MIN = 0
    LOSS_MAX = 2
    JITTER_MIN = 0
    JITTER_MAX = 0
    NUM_TESTS = 100  # have a large view
    NUM_ITERATIONS = 5  # should be > 20, since loss occur very rarely

    if mode == 1:
        FSIZES = ['4M']
        CONCURRENCY = 1
    elif mode == 2:
        FSIZES = ['1024k']
        CONCURRENCY = 5

doe_grid = lhs(5, samples=NUM_TESTS, criterion='center')


if __name__ == "__main__":
    sys.stdout = StdoutLogger()

    banner("max test time: {} sec".format(NUM_ITERATIONS * NUM_TESTS * TIMEOUT))

    banner("system description")
    describe_system()

    banner("prerequisites")
    check("openvpn", "which openvpn")
    check("use cubic", "sysctl net/ipv4/tcp_congestion_control | grep cubic")
    check("busybox", "which busybox")
    check("apache benchmark", "which ab")
    for f in FSIZES:
        check("file of size {} exits in files".format(f), "ls files/{}".format(f))

    banner("build network")
    net = make_network()
    net.get('h2').cmd('busybox httpd -f -p 80 &')

    h1, h2 = net.get('h1'), net.get('h2')

    for bw_mul, fsize_mul, loss_mul, delay_mul, jitter_mul in doe_grid:
        bw = int(BW_MIN + bw_mul * (BW_MAX - BW_MIN))
        fsize = FSIZES[int(len(FSIZES) * fsize_mul)]
        loss = int(100 * (LOSS_MIN + loss_mul * (LOSS_MAX - LOSS_MIN))) / 100.0
        delay = int(DELAY_MIN + delay_mul * (DELAY_MAX - DELAY_MIN))
        jitter = int(JITTER_MIN + jitter_mul * (JITTER_MAX - JITTER_MIN))

        banner('(bw, fsize, loss, delay, jitter) {} {} {} {} {}'.format(bw, fsize, loss, delay, jitter))
        limit_network(net, str(bw) + 'kbit', str(delay) + 'ms', str(jitter) + 'ms', str(loss) + '%')

        times = {'tcp': [], 'quic': []}
        for i in range(NUM_ITERATIONS):

            # tcp
            res = net.get('h1').cmd('timeout {} ab -n {} -c {} 7.7.7.7/files/{} | grep Total:'.format(TIMEOUT, CONCURRENCY, CONCURRENCY, fsize))
            res = res.replace('Total:', '').strip()
            try:
                times['tcp'].append(float(res.split()[1]))
            except:
                times['tcp'].append(-1)

            # quic
            h2.cmd('../quic_bench/quicbench server --pub certificates/server.pub --priv certificates/server &')
            time.sleep(1)
            t0 = time.time()

            res = h1.cmd('timeout {} ../quic_bench/quicbench client -c 7.7.7.7 --streams {} --size {} ; echo $?'.format(TIMEOUT, CONCURRENCY, fsize))
            quic_time = time.time() - t0
            h2.cmd('killall quicbench')
            cmd_status = res.split('\n')[-2].strip()

            if cmd_status != '124':
                times['quic'].append(quic_time * 1000)
            else:
                times['quic'].append(-1)

        print('{};;;{}'.format(
            ';'.join(map(lambda x: str(int(x)), times['tcp'])),
            ';'.join(map(lambda x: str(int(x)), times['quic']))
        ))
        print("")

    banner("clean")
    net.stop()
