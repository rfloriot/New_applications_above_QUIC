# Quic VPN 

A simple VPN built with quic to experiment QUIC as transport protocol for higher level applications. 

## Getting started 

Quic VPN is built to work alike openVPN. 

To build the project, just run a simple: 

    make

And to use it, start the program with the following commands (after editing remote IP): 

    ./quicvpn trial/quic-quicvpn-client.yaml 
    ./quicvpn trial/quic-quicvpn-server.yaml 

Those file should be quite explicit and define most of the possible variables 
for client and server. 

## Assessing performance

In order to compare the performance of this quic VPN with classical tunneling methods, 
we have built a testing framework based on mininet. 

You can launch a mininet instance with `mininet-setup.py`

Then, use the information in the trial README file to perform tests.
