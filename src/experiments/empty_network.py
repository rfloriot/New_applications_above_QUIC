#!/usr/bin/python
from helpers import *

if __name__ == "__main__":
    sys.stdout = StdoutLogger()

    banner("system description")
    describe_system()

    banner("build network")
    net = make_network()
    #make_tunnels(net)
    #disableECN(net)
    #make_port_forwarding(net)


    #limit_network(net, '5000Kbit', '150ms', '0ms', '0%') 
    #limit_network(net, '50000Kbit', '50ms', '0ms', '1%') 
    limit_network(net, '10Mbit', '50ms', '0ms', '0%')
    #limit_network(net, '50Mbit', '10ms', '4ms', '1%')
    #limit_network(net, '1967kbit', '18ms', '0ms', '0.58%')
    #limit_network(net, "8897Kbit", "59ms", "0ms", "1.99%", 100)
    #limit_network(net, "10000Kbit", "100ms", "0ms", "0%", 1000)

    net.interact()

    banner("clean")
    net.stop()
