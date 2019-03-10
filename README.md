- create tun device

For tcp:

- redirect local tcp traffic to tun device
- read tcp packets from tun device
- wrap tcp packets with shadowsocks protocol and sent to server

For udp:

- redirect dns udp packet to tun device
- read udp packets from tun device
- if it's dns query send tcp dns query through shadowsocks protocol(minic ChinaDNS?), return udp response to client
- drop non dns udp packets
