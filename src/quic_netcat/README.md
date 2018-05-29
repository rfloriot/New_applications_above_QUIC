# quicnc 

Port of the "netcat" utility to use the QUIC protocol 

    Usage: quicnc [--listen] [--debug] [--pub PUB] [--priv PRIV] [--req REQ] [--bufsize BUFSIZE] HOST PORT
    
    Positional arguments:
      HOST                   host to contact
      PORT                   port to connect or listen
    
    Options:
      --listen, -l           do we listen or connect?
      --debug, -d            do we set debug mode?
      --pub PUB              server public key
      --priv PRIV            server private key
      --req REQ              remote host public key
      --bufsize BUFSIZE, -b BUFSIZE
                             internal buffer size [default: 200000]
      --help, -h             display this help and exit


example: Without client authentication

    ./quicnc -l localhost 5050 --priv server --pub server.pub
    ./quicnc localhost 5050
    
example: With client authentication 
    
    ./quicnc -l localhost 5050 --priv server --pub server.pub --req client.pub
    ./quicnc localhost 5050 --req server.pub
    
## trial 

In order to assess the difference of performance between NC and QUICNC, you can use the 
`setup.py` file under the `trial/` directory. 